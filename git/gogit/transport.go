/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gogit

import (
	"fmt"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	gossh "golang.org/x/crypto/ssh"

	"github.com/fluxcd/pkg/git"
	"github.com/fluxcd/pkg/ssh/knownhosts"
)

// transportAuth constructs the transport.AuthMethod for the git.Transport of
// the given git.AuthOptions. It returns the result, or an error.
func transportAuth(opts *git.AuthOptions, fallbackToDefaultKnownHosts bool) (transport.AuthMethod, error) {
	if opts == nil {
		return nil, nil
	}
	switch opts.Transport {
	case git.HTTPS, git.HTTP:
		// Some providers (i.e. GitLab) will reject empty credentials for
		// public repositories.
		if opts.Username != "" || opts.Password != "" {
			return &http.BasicAuth{
				Username: opts.Username,
				Password: opts.Password,
			}, nil
		} else if opts.BearerToken != "" {
			return &http.TokenAuth{
				Token: opts.BearerToken,
			}, nil
		}
		return nil, nil
	case git.SSH:
		// if the custom auth options don't provide a private key and known_hosts, we try
		// to use the default known_hosts of the machine.
		if len(opts.Identity)+len(opts.KnownHosts) == 0 && fallbackToDefaultKnownHosts {
			authMethod, err := ssh.DefaultAuthBuilder(opts.Username)
			if err != nil {
				return nil, err
			}
			pkCallback, ok := authMethod.(*ssh.PublicKeysCallback)
			if ok {
				return &DefaultAuth{
					pkCallack: pkCallback,
				}, nil
			}
			return nil, nil
		}
		pk, err := ssh.NewPublicKeys(opts.Username, opts.Identity, opts.Password)
		if err != nil {
			return nil, err
		}

		var callback gossh.HostKeyCallback
		var hkAlgos []string
		if len(opts.KnownHosts) > 0 {
			callback, hkAlgos, err = knownhosts.New(opts.KnownHosts)
			if err != nil {
				return nil, err
			}
		}

		customPK := &CustomPublicKeys{
			pk:       pk,
			callback: callback,
			hkAlgos:  hkAlgos,
		}
		return customPK, nil
	case "":
		return nil, fmt.Errorf("no transport type set")
	default:
		return nil, fmt.Errorf("unknown transport '%s'", opts.Transport)
	}
}

// clientCert returns the client certificate from the given git.AuthOptions.
func clientCert(opts *git.AuthOptions) []byte {
	if opts == nil {
		return nil
	}
	return opts.ClientCert
}

// clientKey returns the client key from the given git.AuthOptions.
func clientKey(opts *git.AuthOptions) []byte {
	if opts == nil {
		return nil
	}
	return opts.ClientKey
}

// caBundle returns the CA bundle from the given git.AuthOptions.
func caBundle(opts *git.AuthOptions) []byte {
	if opts == nil {
		return nil
	}
	return opts.CAFile
}

// CustomPublicKeys is a wrapper around ssh.PublicKeys to help us
// customize the ssh config. It implements ssh.AuthMethod.
type CustomPublicKeys struct {
	pk       *ssh.PublicKeys
	callback gossh.HostKeyCallback
	hkAlgos  []string
}

func (a *CustomPublicKeys) Name() string {
	return a.pk.Name()
}

func (a *CustomPublicKeys) String() string {
	return a.pk.String()
}

func (a *CustomPublicKeys) ClientConfig() (*gossh.ClientConfig, error) {
	if a.callback != nil {
		a.pk.HostKeyCallback = a.callback
	}

	config, err := a.pk.ClientConfig()
	if err != nil {
		return nil, err
	}

	if len(git.KexAlgos) > 0 {
		config.Config.KeyExchanges = git.KexAlgos
	}

	// Whenever ssh-rsa is being used, prioritise sha2-512 and sha2-256.
	// TODO: deprecate SHA1 signing scheme.
	if len(a.hkAlgos) == 1 && a.hkAlgos[0] == "ssh-rsa" {
		a.hkAlgos = append([]string{"rsa-sha2-512", "rsa-sha2-256"}, a.hkAlgos[0])
	}

	config.HostKeyAlgorithms = a.hkAlgos
	if len(git.HostKeyAlgos) > 0 {
		config.HostKeyAlgorithms = git.HostKeyAlgos
	}

	return config, nil
}

type DefaultAuth struct {
	pkCallack *ssh.PublicKeysCallback
}

func (a *DefaultAuth) Name() string {
	return a.pkCallack.Name()
}

func (a *DefaultAuth) String() string {
	return a.pkCallack.String()
}

func (a *DefaultAuth) ClientConfig() (*gossh.ClientConfig, error) {
	config, err := a.pkCallack.ClientConfig()
	if err != nil {
		return nil, err
	}
	return config, nil
}
