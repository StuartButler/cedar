package operations

import (
	"testing"

	"github.com/evergreen-ci/cedar"
	"github.com/stretchr/testify/assert"
)

func TestServiceConfiguration(t *testing.T) {
	for name, test := range map[string]func(t *testing.T, env cedar.Environment){
		"VerifyFixtures": func(t *testing.T, env cedar.Environment) {
			assert.NotNil(t, env)
		},
		"PanicsWithNilEnv": func(t *testing.T, env cedar.Environment) {
			assert.Panics(t, func() {
				_ = configure(nil, 2, true, "foo", "bar", "baz")
			})
		},
		"ErrorsWithInvalidConfigDatabase": func(t *testing.T, env cedar.Environment) {
			err := configure(env, 2, true, "", "", "")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "problem setting up config")
			assert.Contains(t, err.Error(), "mongodb")
		},
		"ErrorsWithInvalidConfigWorkers": func(t *testing.T, env cedar.Environment) {
			err := configure(env, -1, true, "mongodb://localhost:27017", "", "")
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "problem setting up config")
			assert.Contains(t, err.Error(), "workers")
		},
		"ValidOptions": func(t *testing.T, env cedar.Environment) {
			err := configure(env, 2, true, "mongodb://localhost:27017", "foo", "cedar_test")
			assert.NoError(t, err)
		},
		"ConfigurationOfLogging": func(t *testing.T, env cedar.Environment) {
			t.Skip("skipping because the code is improperly factored to support testing")
		},
		// "": func(t *testing.T, env cedar.Environment) {},
	} {
		t.Run(name, func(t *testing.T) {
			env := cedar.GetEnvironment()
			test(t, env)
		})
	}
}
