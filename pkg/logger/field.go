// Copyright 2022 The Corazawaf Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logger

import "go.uber.org/zap"

var (
	// Skip constructs a no-op field, which is often useful when handling invalid
	// inputs in other Field constructors.
	Skip = zap.Skip
	// Binary constructs a field that carries an opaque binary blob.
	//
	// Binary data is serialized in an encoding-appropriate format. For example,
	// zap's JSON encoder base64-encodes binary blobs. To log UTF-8 encoded text,
	// use ByteString.
	Binary = zap.Binary
	// Bool constructs a field that carries a bool.
	Bool = zap.Bool
	// Boolp constructs a field that carries a *bool. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Boolp = zap.Boolp
	// ByteString constructs a field that carries UTF-8 encoded text as a []byte.
	// To log opaque binary blobs (which aren't necessarily valid UTF-8), use
	// Binary.
	ByteString = zap.ByteString
	// Complex128 constructs a field that carries a complex number. Unlike most
	// numeric fields, this costs an allocation (to convert the complex128 to
	// interface{}).
	Complex128 = zap.Complex128
	// Complex128p constructs a field that carries a *complex128. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Complex128p = zap.Complex128p
	// Complex64 constructs a field that carries a complex number. Unlike most
	// numeric fields, this costs an allocation (to convert the complex64 to
	// interface{}).
	Complex64 = zap.Complex64
	// Complex64p constructs a field that carries a *complex64. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Complex64p = zap.Complex64p
	// Float64 constructs a field that carries a float64. The way the
	// floating-point value is represented is encoder-dependent, so marshaling is
	// necessarily lazy.
	Float64 = zap.Float64
	// Float64p constructs a field that carries a *float64. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Float64p = zap.Float64p
	// Float32 constructs a field that carries a float32. The way the
	// floating-point value is represented is encoder-dependent, so marshaling is
	// necessarily lazy.
	Float32 = zap.Float32
	// Float32p constructs a field that carries a *float32. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Float32p = zap.Float32p
	// Int constructs a field with the given key and value.
	Int = zap.Int
	// Intp constructs a field that carries a *int. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Intp = zap.Intp
	// Int64 constructs a field with the given key and value.
	Int64 = zap.Int64
	// Int64p constructs a field that carries a *int64. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Int64p = zap.Int64p
	// Int32 constructs a field with the given key and value.
	Int32 = zap.Int32
	// Int32p constructs a field that carries a *int32. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Int32p = zap.Int32p
	// Int16 constructs a field with the given key and value.
	Int16 = zap.Int16
	// Int16p constructs a field that carries a *int16. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Int16p = zap.Int16p
	// Int8 constructs a field with the given key and value.
	Int8 = zap.Int8
	// Int8p constructs a field that carries a *int8. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Int8p = zap.Int8p
	// String constructs a field with the given key and value.
	String = zap.String
	// Stringp constructs a field that carries a *string. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Stringp = zap.Stringp
	// Uint constructs a field with the given key and value.
	Uint = zap.Uint
	// Uintp constructs a field that carries a *uint. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Uintp = zap.Uintp
	// Uint64 constructs a field with the given key and value.
	Uint64 = zap.Uint64
	// Uint64p constructs a field that carries a *uint64. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Uint64p = zap.Uint64p
	// Uint32 constructs a field with the given key and value.
	Uint32 = zap.Uint32
	// Uint32p constructs a field that carries a *uint32. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Uint32p = zap.Uint32p
	// Uint16 constructs a field with the given key and value.
	Uint16 = zap.Uint16
	// Uint16p constructs a field that carries a *uint16. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Uint16p = zap.Uint16p
	// Uint8 constructs a field with the given key and value.
	Uint8 = zap.Uint8
	// Uint8p constructs a field that carries a *uint8. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Uint8p = zap.Uint8p
	// Uintptr constructs a field with the given key and value.
	Uintptr = zap.Uintptr
	// Uintptrp constructs a field that carries a *uintptr. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Uintptrp = zap.Uintptrp
	// Reflect constructs a field with the given key and an arbitrary object. It uses
	// an encoding-appropriate, reflection-based function to lazily serialize nearly
	// any object into the logging context, but it's relatively slow and
	// allocation-heavy. Outside tests, Any is always a better choice.
	//
	// If encoding fails (e.g., trying to serialize a map[int]string to JSON), Reflect
	// includes the error message in the final log output.
	Reflect = zap.Reflect
	// Namespace creates a named, isolated scope within the logger's context. All
	// subsequent fields will be added to the new namespace.
	//
	// This helps prevent key collisions when injecting loggers into sub-components
	// or third-party libraries.
	Namespace = zap.Namespace
	// Stringer constructs a field with the given key and the output of the value's
	// String method. The Stringer's String method is called lazily.
	Stringer = zap.Stringer
	// Time constructs a Field with the given key and value. The encoder
	// controls how the time is serialized.
	Time = zap.Time
	// Timep constructs a field that carries a *time.Time. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Timep = zap.Timep
	// Stack constructs a field that stores a stacktrace of the current goroutine
	// under provided key. Keep in mind that taking a stacktrace is eager and
	// expensive (relatively speaking); this function both makes an allocation and
	// takes about two microseconds.
	Stack = zap.Stack
	// StackSkip constructs a field similarly to Stack, but also skips the given
	// number of frames from the top of the stacktrace.
	StackSkip = zap.StackSkip
	// Duration constructs a field with the given key and value. The encoder
	// controls how the duration is serialized.
	Duration = zap.Duration
	// Durationp constructs a field that carries a *time.Duration. The returned Field will safely
	// and explicitly represent `nil` when appropriate.
	Durationp = zap.Durationp
	// Object constructs a field with the given key and ObjectMarshaler. It
	// provides a flexible, but still type-safe and efficient, way to add map- or
	// struct-like user-defined types to the logging context. The struct's
	// MarshalLogObject method is called lazily.
	Object = zap.Object
	// Inline constructs a Field that is similar to Object, but it
	// will add the elements of the provided ObjectMarshaler to the
	// current namespace.
	Inline = zap.Inline
	// Any takes a key and an arbitrary value and chooses the best way to represent
	// them as a field, falling back to a reflection-based approach only if
	// necessary.
	//
	// Since byte/uint8 and rune/int32 are aliases, Any can't differentiate between
	// them. To minimize surprises, []byte values are treated as binary blobs, byte
	// values are treated as uint8, and runes are always treated as integers.
	Any = zap.Any
)
