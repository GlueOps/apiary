package proxmox

import (
	"context"
	"errors"
	"fmt"
)

func (cl *Cluster) Notifications(ctx context.Context) (entries ClusterNotificationIndex, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications", &entries)
	return
}

func (cl *Cluster) NotificationMatcherFields(ctx context.Context) (fields []*ClusterNotificationMatcherField, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/matcher-fields", &fields)
	return
}

func (cl *Cluster) NotificationMatcherFieldValues(ctx context.Context) (values []*ClusterNotificationMatcherFieldValue, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/matcher-field-values", &values)
	return
}

func (cl *Cluster) NotificationTargets(ctx context.Context) (targets []*ClusterNotificationTarget, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/targets", &targets)
	return
}

func (cl *Cluster) TestNotificationTarget(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("notification target name can not be empty")
	}
	return cl.client.Post(ctx, fmt.Sprintf("/cluster/notifications/targets/%s/test", name), nil, nil)
}

func (cl *Cluster) NotificationMatchers(ctx context.Context) (matchers []*ClusterNotificationMatcher, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/matchers", &matchers)
	return
}

func (cl *Cluster) NotificationMatcher(ctx context.Context, name string) (m *ClusterNotificationMatcher, err error) {
	if name == "" {
		err = errors.New("notification matcher name can not be empty")
		return
	}
	m = &ClusterNotificationMatcher{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/notifications/matchers/%s", name), m); err != nil {
		return
	}
	if m.Name == "" {
		m.Name = name
	}
	return
}

func (cl *Cluster) NewNotificationMatcher(ctx context.Context, opts *ClusterNotificationMatcherOptions) error {
	if opts == nil || opts.Name == "" {
		return errors.New("notification matcher name can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/notifications/matchers", opts, nil)
}

func (cl *Cluster) UpdateNotificationMatcher(ctx context.Context, name string, opts *ClusterNotificationMatcherOptions) error {
	if name == "" {
		return errors.New("notification matcher name can not be empty")
	}
	if opts == nil {
		opts = &ClusterNotificationMatcherOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/notifications/matchers/%s", name), opts, nil)
}

func (cl *Cluster) DeleteNotificationMatcher(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("notification matcher name can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/notifications/matchers/%s", name), nil)
}

func (cl *Cluster) NotificationGotifyEndpoints(ctx context.Context) (endpoints []*ClusterNotificationGotifyEndpoint, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/endpoints/gotify", &endpoints)
	return
}

func (cl *Cluster) NotificationGotifyEndpoint(ctx context.Context, name string) (e *ClusterNotificationGotifyEndpoint, err error) {
	if name == "" {
		err = errors.New("gotify endpoint name can not be empty")
		return
	}
	e = &ClusterNotificationGotifyEndpoint{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/notifications/endpoints/gotify/%s", name), e); err != nil {
		return
	}
	if e.Name == "" {
		e.Name = name
	}
	return
}

func (cl *Cluster) NewNotificationGotifyEndpoint(ctx context.Context, opts *ClusterNotificationGotifyOptions) error {
	if opts == nil || opts.Name == "" {
		return errors.New("gotify endpoint name can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/notifications/endpoints/gotify", opts, nil)
}

func (cl *Cluster) UpdateNotificationGotifyEndpoint(ctx context.Context, name string, opts *ClusterNotificationGotifyOptions) error {
	if name == "" {
		return errors.New("gotify endpoint name can not be empty")
	}
	if opts == nil {
		opts = &ClusterNotificationGotifyOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/notifications/endpoints/gotify/%s", name), opts, nil)
}

func (cl *Cluster) DeleteNotificationGotifyEndpoint(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("gotify endpoint name can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/notifications/endpoints/gotify/%s", name), nil)
}

func (cl *Cluster) NotificationSendmailEndpoints(ctx context.Context) (endpoints []*ClusterNotificationSendmailEndpoint, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/endpoints/sendmail", &endpoints)
	return
}

func (cl *Cluster) NotificationSendmailEndpoint(ctx context.Context, name string) (e *ClusterNotificationSendmailEndpoint, err error) {
	if name == "" {
		err = errors.New("sendmail endpoint name can not be empty")
		return
	}
	e = &ClusterNotificationSendmailEndpoint{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/notifications/endpoints/sendmail/%s", name), e); err != nil {
		return
	}
	if e.Name == "" {
		e.Name = name
	}
	return
}

func (cl *Cluster) NewNotificationSendmailEndpoint(ctx context.Context, opts *ClusterNotificationSendmailOptions) error {
	if opts == nil || opts.Name == "" {
		return errors.New("sendmail endpoint name can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/notifications/endpoints/sendmail", opts, nil)
}

func (cl *Cluster) UpdateNotificationSendmailEndpoint(ctx context.Context, name string, opts *ClusterNotificationSendmailOptions) error {
	if name == "" {
		return errors.New("sendmail endpoint name can not be empty")
	}
	if opts == nil {
		opts = &ClusterNotificationSendmailOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/notifications/endpoints/sendmail/%s", name), opts, nil)
}

func (cl *Cluster) DeleteNotificationSendmailEndpoint(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("sendmail endpoint name can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/notifications/endpoints/sendmail/%s", name), nil)
}

func (cl *Cluster) NotificationSMTPEndpoints(ctx context.Context) (endpoints []*ClusterNotificationSMTPEndpoint, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/endpoints/smtp", &endpoints)
	return
}

func (cl *Cluster) NotificationSMTPEndpoint(ctx context.Context, name string) (e *ClusterNotificationSMTPEndpoint, err error) {
	if name == "" {
		err = errors.New("smtp endpoint name can not be empty")
		return
	}
	e = &ClusterNotificationSMTPEndpoint{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/notifications/endpoints/smtp/%s", name), e); err != nil {
		return
	}
	if e.Name == "" {
		e.Name = name
	}
	return
}

func (cl *Cluster) NewNotificationSMTPEndpoint(ctx context.Context, opts *ClusterNotificationSMTPOptions) error {
	if opts == nil || opts.Name == "" {
		return errors.New("smtp endpoint name can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/notifications/endpoints/smtp", opts, nil)
}

func (cl *Cluster) UpdateNotificationSMTPEndpoint(ctx context.Context, name string, opts *ClusterNotificationSMTPOptions) error {
	if name == "" {
		return errors.New("smtp endpoint name can not be empty")
	}
	if opts == nil {
		opts = &ClusterNotificationSMTPOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/notifications/endpoints/smtp/%s", name), opts, nil)
}

func (cl *Cluster) DeleteNotificationSMTPEndpoint(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("smtp endpoint name can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/notifications/endpoints/smtp/%s", name), nil)
}

func (cl *Cluster) NotificationWebhookEndpoints(ctx context.Context) (endpoints []*ClusterNotificationWebhookEndpoint, err error) {
	err = cl.client.Get(ctx, "/cluster/notifications/endpoints/webhook", &endpoints)
	return
}

func (cl *Cluster) NotificationWebhookEndpoint(ctx context.Context, name string) (e *ClusterNotificationWebhookEndpoint, err error) {
	if name == "" {
		err = errors.New("webhook endpoint name can not be empty")
		return
	}
	e = &ClusterNotificationWebhookEndpoint{}
	if err = cl.client.Get(ctx, fmt.Sprintf("/cluster/notifications/endpoints/webhook/%s", name), e); err != nil {
		return
	}
	if e.Name == "" {
		e.Name = name
	}
	return
}

func (cl *Cluster) NewNotificationWebhookEndpoint(ctx context.Context, opts *ClusterNotificationWebhookOptions) error {
	if opts == nil || opts.Name == "" {
		return errors.New("webhook endpoint name can not be empty")
	}
	return cl.client.Post(ctx, "/cluster/notifications/endpoints/webhook", opts, nil)
}

func (cl *Cluster) UpdateNotificationWebhookEndpoint(ctx context.Context, name string, opts *ClusterNotificationWebhookOptions) error {
	if name == "" {
		return errors.New("webhook endpoint name can not be empty")
	}
	if opts == nil {
		opts = &ClusterNotificationWebhookOptions{}
	}
	return cl.client.Put(ctx, fmt.Sprintf("/cluster/notifications/endpoints/webhook/%s", name), opts, nil)
}

func (cl *Cluster) DeleteNotificationWebhookEndpoint(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("webhook endpoint name can not be empty")
	}
	return cl.client.Delete(ctx, fmt.Sprintf("/cluster/notifications/endpoints/webhook/%s", name), nil)
}
