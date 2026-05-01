package handlers

import (
	"encoding/json"
	"net/http"

	api "github.com/kagent-dev/kagent/go/api/httpapi"
	"github.com/kagent-dev/kagent/go/api/v1alpha2"
	"github.com/kagent-dev/kagent/go/core/internal/httpserver/errors"
	"github.com/kagent-dev/kagent/go/core/pkg/auth"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// RemoteAgentsHandler handles RemoteAgent CRUD requests.
type RemoteAgentsHandler struct {
	*Base
}

// NewRemoteAgentsHandler creates a new RemoteAgentsHandler.
func NewRemoteAgentsHandler(base *Base) *RemoteAgentsHandler {
	return &RemoteAgentsHandler{Base: base}
}

// HandleListRemoteAgents handles GET /api/remoteagents.
func (h *RemoteAgentsHandler) HandleListRemoteAgents(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("remoteagents-handler").WithValues("operation", "list")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "RemoteAgent"}); err != nil {
		w.RespondWithError(err)
		return
	}

	list := &v1alpha2.RemoteAgentList{}
	if err := h.KubeClient.List(r.Context(), list); err != nil {
		w.RespondWithError(errors.NewInternalServerError("Failed to list RemoteAgents", err))
		return
	}

	log.Info("Listed RemoteAgents", "count", len(list.Items))
	data := api.NewResponse(list.Items, "Successfully listed RemoteAgents", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleCreateRemoteAgent handles POST /api/remoteagents.
func (h *RemoteAgentsHandler) HandleCreateRemoteAgent(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("remoteagents-handler").WithValues("operation", "create")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "RemoteAgent"}); err != nil {
		w.RespondWithError(err)
		return
	}

	var remoteAgent v1alpha2.RemoteAgent
	if err := json.NewDecoder(r.Body).Decode(&remoteAgent); err != nil {
		w.RespondWithError(errors.NewBadRequestError("Invalid RemoteAgent body", err))
		return
	}

	if remoteAgent.Namespace == "" {
		remoteAgent.Namespace = h.DefaultModelConfig.Namespace
	}

	if err := h.KubeClient.Create(r.Context(), &remoteAgent); err != nil {
		if apierrors.IsAlreadyExists(err) {
			w.RespondWithError(errors.NewConflictError("RemoteAgent already exists", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to create RemoteAgent", err))
		return
	}

	log.Info("Created RemoteAgent", "name", remoteAgent.Name, "namespace", remoteAgent.Namespace)
	data := api.NewResponse(remoteAgent, "Successfully created RemoteAgent", false)
	RespondWithJSON(w, http.StatusCreated, data)
}

// HandleUpdateRemoteAgent handles PUT /api/remoteagents/{namespace}/{name}.
func (h *RemoteAgentsHandler) HandleUpdateRemoteAgent(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("remoteagents-handler").WithValues("operation", "update")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "RemoteAgent"}); err != nil {
		w.RespondWithError(err)
		return
	}

	var remoteAgent v1alpha2.RemoteAgent
	if err := json.NewDecoder(r.Body).Decode(&remoteAgent); err != nil {
		w.RespondWithError(errors.NewBadRequestError("Invalid RemoteAgent body", err))
		return
	}

	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("RemoteAgent name is required", err))
		return
	}

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		namespace = h.DefaultModelConfig.Namespace
	}

	remoteAgent.Name = name
	remoteAgent.Namespace = namespace

	if err := h.KubeClient.Update(r.Context(), &remoteAgent); err != nil {
		if apierrors.IsNotFound(err) {
			w.RespondWithError(errors.NewNotFoundError("RemoteAgent not found", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to update RemoteAgent", err))
		return
	}

	log.Info("Updated RemoteAgent", "name", name, "namespace", namespace)
	data := api.NewResponse(remoteAgent, "Successfully updated RemoteAgent", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleGetRemoteAgent handles GET /api/remoteagents/{namespace}/{name}.
func (h *RemoteAgentsHandler) HandleGetRemoteAgent(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("remoteagents-handler").WithValues("operation", "get")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "RemoteAgent"}); err != nil {
		w.RespondWithError(err)
		return
	}

	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("RemoteAgent name is required", err))
		return
	}

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		namespace = h.DefaultModelConfig.Namespace
	}

	remoteAgent := &v1alpha2.RemoteAgent{}
	if err := h.KubeClient.Get(r.Context(), client.ObjectKey{Namespace: namespace, Name: name}, remoteAgent); err != nil {
		if apierrors.IsNotFound(err) {
			w.RespondWithError(errors.NewNotFoundError("RemoteAgent not found", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to get RemoteAgent", err))
		return
	}

	log.Info("Retrieved RemoteAgent", "name", name, "namespace", namespace)
	data := api.NewResponse(remoteAgent, "Successfully retrieved RemoteAgent", false)
	RespondWithJSON(w, http.StatusOK, data)
}

// HandleDeleteRemoteAgent handles DELETE /api/remoteagents/{namespace}/{name}.
func (h *RemoteAgentsHandler) HandleDeleteRemoteAgent(w ErrorResponseWriter, r *http.Request) {
	log := ctrllog.FromContext(r.Context()).WithName("remoteagents-handler").WithValues("operation", "delete")

	if err := Check(h.Authorizer, r, auth.Resource{Type: "RemoteAgent"}); err != nil {
		w.RespondWithError(err)
		return
	}

	name, err := GetPathParam(r, "name")
	if err != nil {
		w.RespondWithError(errors.NewBadRequestError("RemoteAgent name is required", err))
		return
	}

	namespace, err := GetPathParam(r, "namespace")
	if err != nil {
		namespace = h.DefaultModelConfig.Namespace
	}

	remoteAgent := &v1alpha2.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	if err := h.KubeClient.Delete(r.Context(), remoteAgent, &client.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			w.RespondWithError(errors.NewNotFoundError("RemoteAgent not found", err))
			return
		}
		w.RespondWithError(errors.NewInternalServerError("Failed to delete RemoteAgent", err))
		return
	}

	log.Info("Deleted RemoteAgent", "name", name, "namespace", namespace)
	data := api.NewResponse((*v1alpha2.RemoteAgent)(nil), "Successfully deleted RemoteAgent", false)
	RespondWithJSON(w, http.StatusOK, data)
}
