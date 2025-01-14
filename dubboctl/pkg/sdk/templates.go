package sdk

type Templates struct {
	client *Client
}

func newTemplates(client *Client) *Templates {
	return &Templates{client: client}
}
