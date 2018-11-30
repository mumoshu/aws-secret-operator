package awssecret

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"encoding/json"
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

func (c *Context) String(secretId string) (*string, error) {
	if c.s == nil {
		c.s = session.Must(session.NewSession())
	}

	if c.sm == nil {
		c.sm = secretsmanager.New(c.s)
	}

	output, err := c.sm.GetSecretValue(&secretsmanager.GetSecretValueInput{SecretId: &secretId})
	if err != nil {
		return nil, err
	}

	return output.SecretString, nil
}

func (c *Context) JsonAsMap(secretId string) (map[string]string, error) {
	sec, err := c.String(secretId)
	if err != nil {
		return nil, err
	}
	m := map[string]string{}
	if err := json.Unmarshal([]byte(*sec), &m); err != nil {
		return nil, err
	}
	return m, nil
}
