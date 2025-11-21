package scanner

import (
	"context"
	cortexContext "cortex/context"
	"cortex/logging"
	"cortex/repository"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projectdiscovery/naabu/v2/pkg/protocol"
	"github.com/projectdiscovery/naabu/v2/pkg/result"
	"github.com/projectdiscovery/naabu/v2/pkg/runner"
)

type NaabuScanner struct {
	logger *slog.Logger
	repo   repository.ScanRepository
	pool   *pgxpool.Pool
}

func (d NaabuScanner) Scan(ctx context.Context, scan repository.ScanExecution, config repository.ScanConfiguration) error {
	var hosts []string
	for _, asset := range scan.Assets {
		hosts = append(hosts, asset.Endpoint)
	}

	if len(hosts) == 0 {
		d.logger.InfoContext(ctx, "no hosts to scan", logging.FieldScanID, scan.ID)
		return nil
	}
	d.logger.InfoContext(ctx, fmt.Sprintf("starting discovery scan on %d targets", len(hosts)),
		logging.FieldScanID, scan.ID)

	var results []*result.HostResult
	options := runner.Options{
		ScanType: "s",
		Silent:   true,
		Host:     hosts,
		OnResult: func(hostResult *result.HostResult) {
			d.logger.DebugContext(ctx,
				fmt.Sprintf("found %d open ports on %s", len(hostResult.Ports), hostResult.Host),
				logging.FieldScanID, scan.ID)
			results = append(results, hostResult)
		},
		OnReceive: func(hostResult *result.HostResult) {
			// do nothing to prevent logging
		},
	}

	naabu, err := runner.NewRunner(&options)
	if err != nil {
		return err
	}
	defer func(naabu *runner.Runner) {
		_ = naabu.Close()
	}(naabu)

	err = naabu.RunEnumeration(ctx)
	if err != nil {
		return err
	}
	d.logger.InfoContext(ctx, "finished discovery scan", logging.FieldScanID, scan.ID)
	d.logger.InfoContext(ctx, fmt.Sprintf("found %d open ports", len(results)), logging.FieldScanID, scan.ID)

	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}

		err = tx.Commit(ctx)
	}()

	// add changes to database
	now := time.Now()
	for _, naabuResult := range results {
		for _, port := range naabuResult.Ports {
			proto := ProtocolUDP
			if port.Protocol == protocol.TCP {
				proto = ProtocolTCP
			}

			// get current asset findings for this host
			assetID := d.findAssetID(scan.Assets, naabuResult.Host)
			currentFindings, err := d.repo.ListAssetFindings(ctx, tx, assetID)
			if err != nil {
				d.logger.ErrorContext(ctx, "failed to list asset findings",
					logging.FieldScanID, scan.ID, logging.FieldAssetID, assetID, logging.FieldError, err)
				return err
			}

			finding, ok := d.findingExists(currentFindings, port.Port, proto)
			if ok {
				finding.LastSeen = now
			} else {
				finding = &repository.AssetFinding{
					ID:        uuid.New().String(),
					AssetID:   assetID,
					Type:      repository.FindingTypePort,
					FirstSeen: now,
					LastSeen:  now,
				}
				AttachPortData(finding, port.Port, proto)
			}

			err = d.repo.PutAssetFinding(ctx, tx, *finding)
			if err != nil {
				d.logger.ErrorContext(ctx, "failed to put asset discovery result",
					logging.FieldScanID, scan.ID, logging.FieldAssetID, finding.AssetID, logging.FieldError, err)
				return err
			}
		}
	}

	// update scan status
	scan.Status = repository.ScanStatusComplete
	scan.EndTime.Time = now
	err = d.repo.UpdateScan(ctx, tx, scan)
	if err != nil {
		d.logger.ErrorContext(ctx, "failed to update scan",
			logging.FieldScanID, scan.ID,
			logging.FieldError, err)
		return err
	}
	d.logger.DebugContext(ctx, "marked scan complete", logging.FieldScanID, scan.ID)

	// update asset history for finished scan
	userInfo, err := cortexContext.UserInfo(ctx)
	if err != nil {
		d.logger.ErrorContext(ctx, "failed to get user info from context", logging.FieldError, err)
		return err
	}

	for _, asset := range scan.Assets {
		event := repository.AssetHistoryEntry{
			ID:      uuid.New().String(),
			AssetID: asset.ID,
			UserID:  userInfo.UserID,
			Time:    scan.EndTime.Time,
			Type:    repository.ScanAssetEventTypeScanEnded,
			Data: map[string]any{
				"scanId": scan.ID,
				"status": scan.Status,
				"type":   config.Type,
			},
		}

		err = d.repo.AddAssetHistoryEntry(ctx, tx, event)
		if err != nil {
			d.logger.ErrorContext(ctx, "failed to add asset history entry", logging.FieldError, err)
			return err
		}
	}

	return nil
}

func (d NaabuScanner) findAssetID(assets []repository.ScanAsset, endpoint string) string {
	for _, asset := range assets {
		if asset.Endpoint == endpoint {
			return asset.ID
		}
	}
	return ""
}

func (d NaabuScanner) findingExists(findings []repository.AssetFinding, port int, proto Protocol) (*repository.AssetFinding, bool) {
	for _, finding := range findings {
		findingPort := int(finding.Data["port"].(float64))
		findingProto := Protocol(finding.Data["protocol"].(string))

		if finding.Type == repository.FindingTypePort && findingPort == port && findingProto == proto {
			return &finding, true
		}
	}
	return nil, false
}

func NewNaabuScanner(repo repository.ScanRepository, pool *pgxpool.Pool) Scanner {
	return NaabuScanner{
		logger: logging.GetLogger(logging.Scan),
		repo:   repo,
		pool:   pool,
	}
}
