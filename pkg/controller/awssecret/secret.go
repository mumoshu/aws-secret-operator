package awssecret

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"encoding/json"
	"github.com/mumoshu/aws-secret-operator/pkg/apis/mumoshu/v1alpha1"
)

type Context struct {
	s *session.Session
	sm *secretsmanager.SecretsManager
}

func newContext(s *session.Session) *Context {
	return &Context{
		s: s,
	}
}

func (c *Context) String(secretId string, versionId string) (*string, error) {
	if c.s == nil {
		c.s = session.Must(session.NewSession())
	}

	if c.sm == nil {
		c.sm = secretsmanager.New(c.s)
	}

	getSecInput := &secretsmanager.GetSecretValueInput{
		SecretId: &secretId,
		VersionId: &versionId,
	}

	output, err := c.sm.GetSecretValue(getSecInput)
	if err != nil {
		return nil, err
	}

	return output.SecretString, nil
}

func (c *Context) SecretsManagerSecretToKubernetesStringData(ref v1alpha1.SecretsManagerSecretRef) (map[string]string, error) {
	sec, err := c.String(ref.SecretId, ref.VersionId)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	if err := json.Unmarshal([]byte(*sec), &m); err != nil {
		return nil, err
	}
	return m, nil
}
