// Copyright 2013, Cong Ding. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// author: Cong Ding <dinggnu@gmail.com>

package logging

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"time"
)

// watcher watches the logger.queue channel, and writes the logs to output
func (logger *Logger) watcher() {
	var buf bytes.Buffer
	for {
		timeout := time.After(time.Millisecond * logger.timeInterval)

		for i := 0; i < logger.bufferSize; i++ {
			select {
			case msg := <-logger.queue:
				fmt.Fprintln(&buf, msg)
			case message := <-logger.request:
				logger.flushMsg(message)
			case <-timeout:
				i = logger.bufferSize
			case <-logger.flush:
				logger.flushBuf(&buf)
				logger.finish <- true
				i = logger.bufferSize
			case <-logger.quit:
				// If quit signal received, cleans the channel
				// and writes all of them to io.Writer.
				for {
					select {
					case msg := <-logger.queue:
						fmt.Fprintln(&buf, msg)
					case message := <-logger.request:
						logger.flushMsg(message)
					case <-logger.flush:
						// do nothing
					default:
						logger.flushBuf(&buf)
						logger.quit <- true
						return
					}
				}
			}
		}
		logger.flushBuf(&buf)
	}
}

// flushBuf flushes the content of buffer to out and reset the buffer
func (logger *Logger) flushBuf(b *bytes.Buffer) {
	if len(b.Bytes()) > 0 {
		logger.out.Write(b.Bytes())
		b.Reset()
	}
}

// flushMsg is to print log to file, stdout, or others.
func (logger *Logger) flushMsg(message string) {
	if logger.sync {
		logger.wlock.Lock()
		defer logger.wlock.Unlock()
		fmt.Fprintln(logger.out, message)
	} else {
		logger.queue <- message
	}
}

// log records log v... with level `level'.
func (logger *Logger) log(level Level, v ...interface{}) {
	if int32(level) >= atomic.LoadInt32((*int32)(&logger.level)) {
		message := fmt.Sprint(v...)
		message = logger.genLog(level, message)
		if logger.runtime || logger.sync {
			logger.flushMsg(message)
		} else {
			logger.request <- message
		}
	}
}

// logf records log v... with level `level'.
func (logger *Logger) logf(level Level, format string, v ...interface{}) {
	if int32(level) >= atomic.LoadInt32((*int32)(&logger.level)) {
		message := fmt.Sprintf(format, v...)
		message = logger.genLog(level, message)
		if logger.runtime || logger.sync {
			logger.flushMsg(message)
		} else {
			logger.request <- message
		}
	}
}
