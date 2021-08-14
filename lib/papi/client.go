package papi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Resource is an interface to identify a resource
type Resource interface {
	uri() (string, error)
	getDetails(c *Client) error
}

// Downloadable is an interface for a downloadable content
type Downloadable interface {
	Resource
	downloadURL() (string, error)
}

type basicAuth struct {
	username string
	password string
}

// Client of the polochon API
type Client struct {
	endpoint  string
	token     string
	basicAuth *basicAuth
}

// New returns a new Client, url is the base url of
// your polochon API
func New(endpoint string) (*Client, error) {
	if _, err := url.Parse(endpoint); err != nil {
		return nil, err
	}

	return &Client{
		endpoint: endpoint,
	}, nil
}

// SetToken sets the token
func (c *Client) SetToken(token string) {
	c.token = token
}

// SetBasicAuth set the basic auth details
func (c *Client) SetBasicAuth(username, password string) {
	c.basicAuth = &basicAuth{
		username: username,
		password: password,
	}
}

// GetDetails get the detailed informations of a resource
func (c *Client) GetDetails(resource Resource) error {
	return resource.getDetails(c)
}

// DownloadURL returns the download URL of a downloadable content
func (c *Client) DownloadURL(target Downloadable) (string, error) {
	url, err := target.downloadURL()
	if err != nil {
		return "", err
	}

	return c.endpoint + "/" + url, nil
}

// DownloadURLWithToken returns the url with the token
func (c *Client) DownloadURLWithToken(target Downloadable) (string, error) {
	url, err := c.DownloadURL(target)
	if err != nil {
		return "", err
	}

	if c.token != "" {
		url += "?token=" + c.token
	}

	return url, nil
}

// Delete deletes a ressource
func (c *Client) Delete(target Resource) error {
	url, err := target.uri()
	if err != nil {
		return err
	}

	return c.delete(fmt.Sprintf("%s/%s", c.endpoint, url))
}

func (c *Client) get(url string, result interface{}) error {
	return c.request("GET", url, nil, result)
}

func (c *Client) post(url string, data, result interface{}) error {
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(data)
	if err != nil {
		return err
	}

	return c.request("POST", url, buf, result)
}

func (c *Client) delete(url string) error {
	return c.request("DELETE", url, nil, nil)
}

func (c *Client) request(httpType, url string, data io.Reader, result interface{}) error {
	req, err := http.NewRequest(httpType, url, data)
	if err != nil {
		return err
	}

	if c.token != "" {
		req.Header.Add("X-Auth-Token", c.token)
	}

	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.username, c.basicAuth.password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// sometimes there's nothing in the Body and we don't wanna parse it
		if result == nil {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(&result)
	case http.StatusNotFound:
		// return the not found error
		return ErrResourceNotFound
	default:
		// default polochon error is a JSON with an error field
		polochonErr := struct {
			Error string `json:"error"`
		}{
			Error: "Unknown error",
		}
		// If the decode returns an error we ignore it, the default "Unknown
		// error" message will be returned
		json.NewDecoder(resp.Body).Decode(&polochonErr)
		return fmt.Errorf("papi: HTTP error status %s: %s", resp.Status, polochonErr.Error)
	}
}
