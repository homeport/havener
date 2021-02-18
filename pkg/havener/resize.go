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

package havener

import (
	"os"
	"os/signal"

	"github.com/gonvenience/term"
	"golang.org/x/sys/unix"
	"k8s.io/client-go/tools/remotecommand"
)

type terminalSizeQueue struct {
	resize chan remotecommand.TerminalSize
	end    chan struct{}
}

func setupTerminalResizeWatcher() *terminalSizeQueue {
	size := func() remotecommand.TerminalSize {
		w, h := term.GetTerminalSize()
		return remotecommand.TerminalSize{
			Width:  uint16(w),
			Height: uint16(h),
		}
	}

	tsq := &terminalSizeQueue{
		resize: make(chan remotecommand.TerminalSize, 1),
		end:    make(chan struct{}),
	}

	// initialise channel with terminal size of program start
	tsq.resize <- size()

	// start Go routine to watch for signals of a resize event
	go func() {
		winch := make(chan os.Signal, 1)
		signal.Notify(winch, unix.SIGWINCH)
		defer signal.Stop(winch)

		for {
			select {
			case <-winch:
				select {
				case tsq.resize <- size():
					// ok

				default:
					// not so ok
				}

			case <-tsq.end:
				return
			}
		}
	}()

	return tsq
}

func (s *terminalSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-s.resize
	if !ok {
		return nil
	}

	return &size
}

func (s *terminalSizeQueue) stop() {
	close(s.end)
}
