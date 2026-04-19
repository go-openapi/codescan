package classification

import (
	"testing"

	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

func VerifyInfo(t *testing.T, info *oaispec.Info) {
	t.Helper()

	require.NotNil(t, info)
	assert.EqualT(t, "0.0.1", info.Version)
	assert.EqualT(t, "there are no TOS at this moment, use at your own risk we take no responsibility", info.TermsOfService)
	assert.EqualT(t, "Petstore API.", info.Title)

	const descr = `the purpose of this application is to provide an application
that is using plain go code to define an API

This should demonstrate all the possible comment annotations
that are available to turn go code into a fully compliant swagger 2.0 spec`

	assert.EqualT(t, descr, info.Description)

	require.NotNil(t, info.License)
	assert.EqualT(t, "MIT", info.License.Name)
	assert.EqualT(t, "http://opensource.org/licenses/MIT", info.License.URL)

	require.NotNil(t, info.Contact)
	assert.EqualT(t, "John Doe", info.Contact.Name)
	assert.EqualT(t, "john.doe@example.com", info.Contact.Email)
	assert.EqualT(t, "http://john.doe.com", info.Contact.URL)
}
