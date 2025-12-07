package handler

import (
	"cortex/repository"
	"cortex/service"
	"net/http"
)

type createAgentRequestBody struct {
	Name string `json:"name"`
}

type updateAgentRequestBody struct {
	Name string `json:"name"`
}

type createAgentResponse struct {
	Agent *repository.Agent `json:"agent"`
	Token string            `json:"token"`
}

type AgentHandler struct {
	agentService service.AgentService
}

func NewAgentHandler(agentService service.AgentService) *AgentHandler {
	return &AgentHandler{
		agentService: agentService,
	}
}

func (h AgentHandler) HandleListAgents(w http.ResponseWriter, r *http.Request) error {
	agents, err := h.agentService.ListAgents(r.Context())
	if err != nil {
		return WrapError(err)
	}

	if err = RespondMany(w, r, agents); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AgentHandler) HandleGetAgent(w http.ResponseWriter, r *http.Request) error {
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

	agent, err := h.agentService.GetAgent(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, agent); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AgentHandler) HandleCreateAgent(w http.ResponseWriter, r *http.Request) error {
	var requestBody createAgentRequestBody
	err := ValidateRequestBody(r, &requestBody,
		Field(&requestBody.Name, Required(), Length(1, 255)),
	)
	if err != nil {
		return WrapError(err)
	}

	agent, token, err := h.agentService.CreateAgent(r.Context(), requestBody.Name)
	if err != nil {
		return WrapError(err)
	}

	response := createAgentResponse{
		Agent: agent,
		Token: token,
	}

	if err = RespondOneCreated(w, r, response); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AgentHandler) HandleUpdateAgent(w http.ResponseWriter, r *http.Request) error {
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

	var requestBody updateAgentRequestBody
	err = ValidateRequestBody(r, &requestBody,
		Field(&requestBody.Name, Required(), Length(1, 255)),
	)
	if err != nil {
		return WrapError(err)
	}

	agent, err := h.agentService.UpdateAgent(r.Context(), id, requestBody.Name)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, agent); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AgentHandler) HandleDeleteAgent(w http.ResponseWriter, r *http.Request) error {
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

	agent, err := h.agentService.DeleteAgent(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, agent); err != nil {
		return WrapError(err)
	}
	return nil
}
