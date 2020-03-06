package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pkg/errors"
)
import "github.com/aws/aws-sdk-go/service/wafv2"

type WAFV2WebACL struct {
	svc       *wafv2.WAFV2
	ID        *string
	Name      *string
	Scope     *string
	LockToken *string
}

func init() {
	register("WAFV2WebACL", ListWAFV2WebACLs)
}

func ListWAFV2WebACLs(sess *session.Session) ([]Resource, error) {
	var resources []WAFV2WebACL
	svc := wafv2.New(sess)

	// Lookup resource for both "CLOUDFRONT" and "REGIONAL" scopes
	scopes := []string{"CLOUDFRONT", "REGIONAL"}
	for _, scope := range scopes {
		scopeResources, err := listWAFV2WebACLsForScope(svc, &scope)
		if err != nil {
			return nil, err
		}
		resources = append(resources, scopeResources...)
	}

	return resources, nil
}

func listWAFV2WebACLsForScope(svc *wafv2.WAFV2, scope *string) ([]WAFV2WebACL, error) {
	var resources []WAFV2WebACL

	params := &wafv2.ListWebACLsInput{
		NextMarker: nil,
		Scope:      scope,
	}

	for {
		resp, err := svc.ListWebACLs(params)

		if err != nil {
			return nil, err
		}

		for _, acl := range resp.WebACLs {
			resources = append(resources, WAFV2WebACL{
				ID:        acl.Id,
				Name:      acl.Name,
				Scope:     scope,
				LockToken: acl.LockToken,
			})
		}

		// Pagination is complete
		if resp.NextMarker == nil {
			break
		}

		// Continue pagination
		params.NextMarker = resp.NextMarker
	}

	return resources, nil
}

func (r *WAFV2WebACL) Remove() error {
	_, err := r.svc.DeleteWebACL(&wafv2.DeleteWebACLInput{
		Id:        r.ID,
		LockToken: r.LockToken,
		Name:      r.Name,
		Scope:     r.Scope,
	})

	if err != nil {
		// DeleteWebACL will fail if the resource has changed
		// since we first retrieved our LockToken.
		// In this case, we want to update the LockToken, before trying again
		if IsAWSError(err, wafv2.ErrCodeWAFOptimisticLockException) {
			refreshErr := r.refreshLockToken()
			if refreshErr != nil {
				return errors.Wrap(refreshErr, "Failed to update LockToken after deletion failure")
			}
		}
		return err
	}

	return nil
}

func (r *WAFV2WebACL) refreshLockToken() error {
	resp, err := r.svc.GetWebACL(&wafv2.GetWebACLInput{
		Id:    r.ID,
		Name:  r.Name,
		Scope: r.Scope,
	})
	if err != nil {
		return err
	}

	r.LockToken = resp.LockToken

	return nil
}

func (r *WAFV2WebACL) String() string {
	return *r.Name
}
