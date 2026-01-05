package v1

import (
	"context"
	"net/http"

	"github.com/sagarsuperuser/userprofile/internal/httputil"
)

// GetUserList godoc
//
//	@Summary	Get a list of users
//	@Tags		user
//	@Produce	json
//	@Success	200	{object}	[]store.User	"User list"
//	@Failure	500	{object}	nil				"Failed to fetch user list"
//	@Router		/api/v1/user [GET]
func (s *APIV1Service) GetUserList(ctx context.Context, rw http.ResponseWriter, req *http.Request, vars map[string]string) error {
	// panic("TESTING RECOVERY")}
	return httputil.WriteRawJSON(rw, http.StatusOK, nil)
}
