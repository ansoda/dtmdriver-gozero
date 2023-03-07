package driver

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/zeromicro/zero-contrib/zrpc/registry/consul"

	"github.com/dtm-labs/dtmdriver"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/zrpc/resolver"
)

const (
	DriverName = "dtm-driver-gozero"
	kindEtcd   = "etcd"
	kindDiscov = "discov"
	kindConsul = "consul"
)

type (
	zeroDriver struct{}
)

func (z *zeroDriver) GetName() string {
	return DriverName
}

func (z *zeroDriver) RegisterAddrResolver() {
	resolver.Register()
}

func (z *zeroDriver) RegisterService(target string, endpoint string) error {
	if target == "" { // empty target, no action
		return nil
	}
	u, err := url.Parse(target)
	if err != nil {
		return err
	}

	opts := make([]discov.PubOption, 0)
	query, _ := url.ParseQuery(u.RawQuery)
	if query.Get("user") != "" {
		opts = append(opts, discov.WithPubEtcdAccount(query.Get("user"), query.Get("password")))
	}

	switch u.Scheme {
	case kindDiscov:
		fallthrough
	case kindEtcd:
		pub := discov.NewPublisher(strings.Split(u.Host, ","), strings.TrimPrefix(u.Path, "/"), endpoint, opts...)
		pub.KeepAlive()
	case kindConsul:
		return consul.RegisterService(endpoint, consul.Conf{
			Host: u.Host,
			Key:  strings.TrimPrefix(u.Path, "/"),
			Tag:  []string{"tag", "rpc"},
			Meta: map[string]string{
				"Protocol": "grpc",
			},
		})
	default:
		return fmt.Errorf("unknown scheme: %s", u.Scheme)
	}

	return nil
}

func (z *zeroDriver) ParseServerMethod(uri string) (server string, method string, err error) {
	if !strings.Contains(uri, "//") { // 处理无scheme的情况，如果您没有直连，可以不处理
		sep := strings.IndexByte(uri, '/')
		if sep == -1 {
			return "", "", fmt.Errorf("bad url: '%s'. no '/' found", uri)
		}
		return uri[:sep], uri[sep:], nil

	}
	//resolve gozero consul wait=xx url.Parse no standard
	if (strings.Contains(uri, kindConsul)) && strings.Contains(uri, "?") {
		tmp := strings.Split(uri, "?")
		sep := strings.IndexByte(tmp[1], '/')
		if sep == -1 {
			return "", "", fmt.Errorf("bad url: '%s'. no '/' found", uri)
		}
		uri = tmp[0] + tmp[1][sep:]
	}

	u, err := url.Parse(uri)
	if err != nil {
		return "", "", nil
	}
	index := strings.IndexByte(u.Path[1:], '/') + 1

	return u.Scheme + "://" + u.Host + u.Path[:index], u.Path[index:], nil
}

func init() {
	dtmdriver.Register(&zeroDriver{})
}
