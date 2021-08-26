package authz

import (
	"context"
	"fmt"
	"net/http"

	"github.com/porter-dev/porter/api/server/authz/policy"
	"github.com/porter-dev/porter/api/server/shared"
	"github.com/porter-dev/porter/api/server/shared/apierrors"
	"github.com/porter-dev/porter/api/types"
	"github.com/porter-dev/porter/internal/models"
	"gorm.io/gorm"
)

type InfraScopedFactory struct {
	config *shared.Config
}

func NewInfraScopedFactory(
	config *shared.Config,
) *InfraScopedFactory {
	return &InfraScopedFactory{config}
}

func (p *InfraScopedFactory) Middleware(next http.Handler) http.Handler {
	return &InfraScopedMiddleware{next, p.config}
}

type InfraScopedMiddleware struct {
	next   http.Handler
	config *shared.Config
}

func (p *InfraScopedMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// read the project to check scopes
	proj, _ := r.Context().Value(types.ProjectScope).(*models.Project)

	// get the registry id from the URL param context
	reqScopes, _ := r.Context().Value(RequestScopeCtxKey).(map[types.PermissionScope]*policy.RequestAction)
	infraID := reqScopes[types.InfraScope].Resource.UInt

	infra, err := p.config.Repo.Infra().ReadInfra(proj.ID, infraID)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			apierrors.HandleAPIError(w, p.config.Logger, apierrors.NewErrForbidden(
				fmt.Errorf("infra with id %d not found in project %d", infraID, proj.ID),
			))
		} else {
			apierrors.HandleAPIError(w, p.config.Logger, apierrors.NewErrInternal(err))
		}

		return
	}

	ctx := NewInfraContext(r.Context(), infra)
	r = r.WithContext(ctx)
	p.next.ServeHTTP(w, r)
}

func NewInfraContext(ctx context.Context, infra *models.Infra) context.Context {
	return context.WithValue(ctx, types.InfraScope, infra)
}
