package napcat

import "context"

func (c *napcatClient) parse_ID(ID string) error {
	return nil
}

func (c *napcatClient) auth(userID string) bool {
	if userID == c.autherID {
		return true
	}
	return false
}

func (c *napcatClient) Sendlog(ctx context.Context, message string) error {
	return c.SentPrivateMessage(ctx, c.autherID, message)
}
