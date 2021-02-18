// Copyright © 2021 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"io"

	"github.com/gonvenience/wrap"
)

// duplicateReader creates a given number of io.Reader duplicates based on the
// provided input reader. This way it is possible to use one input reader for
// more than one consumer.
func duplicateReader(reader io.Reader, count int) []io.Reader {
	writers := []io.Writer{}
	readers := []io.Reader{}
	for i := 0; i < count; i++ {
		r, w := io.Pipe()
		writers = append(writers, w)
		readers = append(readers, r)
	}

	writer := io.MultiWriter(writers...)
	go func() {
		if _, err := io.Copy(writer, reader); err != nil {
			panic(err)
		}

		for i := range writers {
			if w, ok := writers[i].(io.Closer); ok {
				w.Close()
			}
		}
	}()

	return readers
}

func combineErrorsFromChannel(context string, c chan error) error {
	errors := []error{}
	for err := range c {
		if err != nil {
			errors = append(errors, err)
		}
	}

	switch len(errors) {
	case 0:
		return nil

	default:
		return wrap.Errors(errors, context)
	}
}
