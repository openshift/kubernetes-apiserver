/*
Copyright 2021 The Kubernetes Authors.

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

package filters

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestRequestContextWithUpperBoundOrWorkAroundOurBrokenCaseWhereTimeoutWasNotAppliedYet(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		context    func(r *http.Request) (context.Context, context.CancelFunc)
		upperBound time.Duration
		remaining  time.Duration
	}{
		{
			name: "request context has a bound deadline",
			url:  "/foo",
			context: func(r *http.Request) (context.Context, context.CancelFunc) {
				return context.WithTimeout(r.Context(), time.Minute)
			},
			upperBound: 30 * time.Second,
			remaining:  28 * time.Second, // to account for flakes in unit test
		},
		{
			name: "request context does not have any bound deadline, no user specified timeout",
			url:  "/foo",
			context: func(r *http.Request) (context.Context, context.CancelFunc) {
				return context.WithCancel(r.Context())
			},
			upperBound: 30 * time.Second,
			remaining:  28 * time.Second, // to account for flakes in unit test

		},
		{
			name: "request context does not have any bound deadline, user specified timeout is malformed",
			url:  "/foo?timeout=invalid",
			context: func(r *http.Request) (context.Context, context.CancelFunc) {
				return context.WithCancel(r.Context())
			},
			upperBound: 30 * time.Second,
			remaining:  28 * time.Second, // to account for flakes in unit test

		},
		{
			name: "request context does not have any bound deadline, user specified timeout is zero",
			url:  "/foo?timeout=0s",
			context: func(r *http.Request) (context.Context, context.CancelFunc) {
				return context.WithCancel(r.Context())
			},
			upperBound: 30 * time.Second,
			remaining:  28 * time.Second, // to account for flakes in unit test

		},
		{
			name: "request context does not have any bound deadline, user specified timeout is valid",
			url:  "/foo?timeout=5m2s",
			context: func(r *http.Request) (context.Context, context.CancelFunc) {
				return context.WithCancel(r.Context())
			},
			upperBound: time.Minute,
			remaining:  5 * time.Minute, // to account for flakes in unit test

		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := newRequest(t, test.url)

			parent, parentCancel := test.context(req)
			defer parentCancel()
			req = req.WithContext(parent)

			ctx, cancel := RequestContextWithUpperBoundOrWorkAroundOurBrokenCaseWhereTimeoutWasNotAppliedYet(req, test.upperBound)
			defer cancel()

			deadline, ok := ctx.Deadline()
			if !ok {
				t.Errorf("Expected the context to have a deadline, but got: %t", ok)
			}

			remainingGot := time.Until(deadline)
			if remainingGot <= test.remaining {
				t.Errorf("Expected the remaining deadline to be greater, wanted: %s, but got: %s", test.remaining, remainingGot)
			}
		})
	}
}
