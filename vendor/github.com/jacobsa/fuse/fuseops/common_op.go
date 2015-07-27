// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fuseops

import (
	"fmt"
	"log"
	"reflect"
	"strings"
	"syscall"

	"github.com/jacobsa/fuse/internal/buffer"
	"github.com/jacobsa/reqtrace"
	"golang.org/x/net/context"
)

// An interface that all ops inside which commonOp is embedded must
// implement.
type internalOp interface {
	Op

	// Create a response message for the kernel, leaving the leading
	// fusekernel.OutHeader untouched.
	//
	// Special case: a zero return value means that the kernel is not expecting a
	// response.
	kernelResponse() (b buffer.OutMessage)
}

// A function that sends a reply message back to the kernel for the request
// with the given fuse unique ID. The error argument is for informational
// purposes only; the error to hand to the kernel is encoded in the message.
type replyFunc func(Op, uint64, []byte, error) error

// A helper for embedding common behavior.
type commonOp struct {
	// The context exposed to the user.
	ctx context.Context

	// The op in which this struct is embedded.
	op internalOp

	// The fuse unique ID of this request, as assigned by the kernel.
	fuseID uint64

	// A function that can be used to send a reply to the kernel.
	sendReply replyFunc

	// A function that can be used to log debug information about the op. The
	// first argument is a call depth.
	//
	// May be nil.
	debugLog func(int, string, ...interface{})

	// A logger to be used for logging exceptional errors.
	//
	// May be nil.
	errorLogger *log.Logger
}

func (o *commonOp) ShortDesc() (desc string) {
	v := reflect.ValueOf(o.op)
	opName := v.Type().String()

	// Attempt to better handle the usual case: a string that looks like
	// "*fuseops.GetInodeAttributesOp".
	const prefix = "*fuseops."
	const suffix = "Op"
	if strings.HasPrefix(opName, prefix) && strings.HasSuffix(opName, suffix) {
		opName = opName[len(prefix) : len(opName)-len(suffix)]
	}

	desc = opName

	// Include the inode number to which the op applies, if possible.
	if f := v.Elem().FieldByName("Inode"); f.IsValid() {
		desc = fmt.Sprintf("%s(inode=%v)", desc, f.Interface())
	}

	return
}

func (o *commonOp) DebugString() string {
	// By default, defer to ShortDesc.
	return o.op.ShortDesc()
}

func (o *commonOp) init(
	ctx context.Context,
	op internalOp,
	fuseID uint64,
	sendReply replyFunc,
	debugLog func(int, string, ...interface{}),
	errorLogger *log.Logger) {
	// Initialize basic fields.
	o.ctx = ctx
	o.op = op
	o.fuseID = fuseID
	o.sendReply = sendReply
	o.debugLog = debugLog
	o.errorLogger = errorLogger

	// Set up a trace span for this op.
	var reportForTrace reqtrace.ReportFunc
	o.ctx, reportForTrace = reqtrace.StartSpan(o.ctx, o.op.ShortDesc())

	// When the op is finished, report to both reqtrace and the connection.
	prevSendReply := o.sendReply
	o.sendReply = func(op Op, fuseID uint64, msg []byte, opErr error) (err error) {
		reportForTrace(opErr)
		err = prevSendReply(op, fuseID, msg, opErr)
		return
	}
}

func (o *commonOp) Context() context.Context {
	return o.ctx
}

func (o *commonOp) Logf(format string, v ...interface{}) {
	if o.debugLog == nil {
		return
	}

	const calldepth = 2
	o.debugLog(calldepth, format, v...)
}

func (o *commonOp) Respond(err error) {
	// If successful, we ask the op for an appopriate response to the kernel, and
	// it is responsible for leaving room for the fusekernel.OutHeader struct.
	// Otherwise, create our own.
	var b buffer.OutMessage
	if err == nil {
		b = o.op.kernelResponse()
	} else {
		b = buffer.NewOutMessage(0)
	}

	// Fill in the header if a reply is needed.
	msg := b.Bytes()
	if msg != nil {
		h := b.OutHeader()
		h.Unique = o.fuseID
		h.Len = uint32(len(msg))
		if err != nil {
			// If the user gave us a syscall.Errno, use that value in the reply.
			// Otherwise use the generic EIO.
			if errno, ok := err.(syscall.Errno); ok {
				h.Error = -int32(errno)
			} else {
				h.Error = -int32(syscall.EIO)
			}
		}
	}

	// Reply.
	replyErr := o.sendReply(o.op, o.fuseID, msg, err)
	if replyErr != nil && o.errorLogger != nil {
		o.errorLogger.Printf("Error from sendReply: %v", replyErr)
	}
}
