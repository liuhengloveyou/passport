// Copyright 2012 The Gorilla Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sessions

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"strconv"
	"time"
)

// Codec defines an interface to encode and decode cookie values.
type Codec interface {
	Encode(name string, value interface{}) (string, error)
	Decode(name, value string, dst interface{}) error
}

// New returns a new SecureCookie.
//
// hashKey is required, used to authenticate values using HMAC. Create it using
// GenerateRandomKey(). It is recommended to use a key with 32 or 64 bytes.
//
// blockKey is optional, used to encrypt values. Create it using
// GenerateRandomKey(). The key length must correspond to the key size
// of the encryption algorithm. For AES, used by default, valid lengths are
// 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256.
// The default encoder used for cookie serialization is encoding/gob.
//
// Note that keys created using GenerateRandomKey() are not automatically
// persisted. New keys will be created when the application is restarted, and
// previously issued cookies will not be able to be decoded.
func NewSecureCookie(hashKey, blockKey []byte) *SecureCookie {
	s := &SecureCookie{
		hashKey:   hashKey,
		blockKey:  blockKey,
		hashFunc:  sha256.New,
		maxAge:    86400 * 30,
		maxLength: 4096,
		sz:        GobEncoder{},
	}
	if len(hashKey) == 0 {
		s.err = errHashKeyNotSet
	}
	if blockKey != nil {
		s.BlockFunc(aes.NewCipher)
	}
	return s
}

// SecureCookie encodes and decodes authenticated and optionally encrypted
// cookie values.
type SecureCookie struct {
	hashKey   []byte
	hashFunc  func() hash.Hash
	blockKey  []byte
	block     cipher.Block
	maxLength int
	maxAge    int64
	minAge    int64
	err       error
	sz        Serializer
	// For testing purposes, the function that returns the current timestamp.
	// If not set, it will use time.Now().UTC().Unix().
	timeFunc func() int64
}

// Serializer provides an interface for providing custom serializers for cookie
// values.
type Serializer interface {
	Serialize(src interface{}) ([]byte, error)
	Deserialize(src []byte, dst interface{}) error
}

// GobEncoder encodes cookie values using encoding/gob. This is the simplest
// encoder and can handle complex types via gob.Register.
type GobEncoder struct{}

// JSONEncoder encodes cookie values using encoding/json. Users who wish to
// encode complex types need to satisfy the json.Marshaller and
// json.Unmarshaller interfaces.
type JSONEncoder struct{}

// NopEncoder does not encode cookie values, and instead simply accepts a []byte
// (as an interface{}) and returns a []byte. This is particularly useful when
// you encoding an object upstream and do not wish to re-encode it.
type NopEncoder struct{}

// MaxLength restricts the maximum length, in bytes, for the cookie value.
//
// Default is 4096, which is the maximum value accepted by Internet Explorer.
func (s *SecureCookie) MaxLength(value int) *SecureCookie {
	s.maxLength = value
	return s
}

// MaxAge restricts the maximum age, in seconds, for the cookie value.
//
// Default is 86400 * 30. Set it to 0 for no restriction.
func (s *SecureCookie) MaxAge(value int) *SecureCookie {
	s.maxAge = int64(value)
	return s
}

// MinAge restricts the minimum age, in seconds, for the cookie value.
//
// Default is 0 (no restriction).
func (s *SecureCookie) MinAge(value int) *SecureCookie {
	s.minAge = int64(value)
	return s
}

// HashFunc sets the hash function used to create HMAC.
//
// Default is crypto/sha256.New.
func (s *SecureCookie) HashFunc(f func() hash.Hash) *SecureCookie {
	s.hashFunc = f
	return s
}

// BlockFunc sets the encryption function used to create a cipher.Block.
//
// Default is crypto/aes.New.
func (s *SecureCookie) BlockFunc(f func([]byte) (cipher.Block, error)) *SecureCookie {
	if s.blockKey == nil {
		s.err = errBlockKeyNotSet
	} else if block, err := f(s.blockKey); err == nil {
		s.block = block
	} else {
		s.err = cookieError{cause: err, typ: usageError}
	}
	return s
}

// Encoding sets the encoding/serialization method for cookies.
//
// Default is encoding/gob.  To encode special structures using encoding/gob,
// they must be registered first using gob.Register().
func (s *SecureCookie) SetSerializer(sz Serializer) *SecureCookie {
	s.sz = sz

	return s
}

// Encode encodes a cookie value.
//
// It serializes, optionally encrypts, signs with a message authentication code,
// and finally encodes the value.
//
// The name argument is the cookie name. It is stored with the encoded value.
// The value argument is the value to be encoded. It can be any value that can
// be encoded using the currently selected serializer; see SetSerializer().
//
// It is the client's responsibility to ensure that value, when encoded using
// the current serialization/encryption settings on s and then base64-encoded,
// is shorter than the maximum permissible length.
func (s *SecureCookie) Encode(name string, value interface{}) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	if s.hashKey == nil {
		s.err = errHashKeyNotSet
		return "", s.err
	}
	var err error
	var b []byte
	// 1. Serialize.
	if b, err = s.sz.Serialize(value); err != nil {
		return "", cookieError{cause: err, typ: usageError}
	}
	// 2. Encrypt (optional).
	if s.block != nil {
		if b, err = encrypt(s.block, b); err != nil {
			return "", cookieError{cause: err, typ: usageError}
		}
	}
	b = encode(b)
	// 3. Create MAC for "name|date|value". Extra pipe to be used later.
	b = []byte(fmt.Sprintf("%s|%d|%s|", name, s.timestamp(), b))
	mac := createMac(hmac.New(s.hashFunc, s.hashKey), b[:len(b)-1])
	// Append mac, remove name.
	b = append(b, mac...)[len(name)+1:]
	// 4. Encode to base64.
	b = encode(b)
	// 5. Check length.
	if s.maxLength != 0 && len(b) > s.maxLength {
		return "", fmt.Errorf("%s: %d", errEncodedValueTooLong, len(b))
	}
	// Done.
	return string(b), nil
}

// Decode decodes a cookie value.
//
// It decodes, verifies a message authentication code, optionally decrypts and
// finally deserializes the value.
//
// The name argument is the cookie name. It must be the same name used when
// it was stored. The value argument is the encoded cookie value. The dst
// argument is where the cookie will be decoded. It must be a pointer.
func (s *SecureCookie) Decode(name, value string, dst interface{}) error {
	if s.err != nil {
		return s.err
	}
	if s.hashKey == nil {
		s.err = errHashKeyNotSet
		return s.err
	}
	// 1. Check length.
	if s.maxLength != 0 && len(value) > s.maxLength {
		return fmt.Errorf("%s: %d", errValueToDecodeTooLong, len(value))
	}
	// 2. Decode from base64.
	b, err := decode([]byte(value))
	if err != nil {
		return err
	}
	// 3. Verify MAC. Value is "date|value|mac".
	parts := bytes.SplitN(b, []byte("|"), 3)
	if len(parts) != 3 {
		return ErrMacInvalid
	}
	h := hmac.New(s.hashFunc, s.hashKey)
	b = append([]byte(name+"|"), b[:len(b)-len(parts[2])-1]...)
	if err = verifyMac(h, b, parts[2]); err != nil {
		return err
	}
	// 4. Verify date ranges.
	var t1 int64
	if t1, err = strconv.ParseInt(string(parts[0]), 10, 64); err != nil {
		return errTimestampInvalid
	}
	t2 := s.timestamp()
	if s.minAge != 0 && t1 > t2-s.minAge {
		return errTimestampTooNew
	}
	if s.maxAge != 0 && t1 < t2-s.maxAge {
		return errTimestampExpired
	}
	// 5. Decrypt (optional).
	b, err = decode(parts[1])
	if err != nil {
		return err
	}
	if s.block != nil {
		if b, err = decrypt(s.block, b); err != nil {
			return err
		}
	}
	// 6. Deserialize.
	if err = s.sz.Deserialize(b, dst); err != nil {
		return cookieError{cause: err, typ: decodeError}
	}
	// Done.
	return nil
}

// timestamp returns the current timestamp, in seconds.
//
// For testing purposes, the function that generates the timestamp can be
// overridden. If not set, it will return time.Now().UTC().Unix().
func (s *SecureCookie) timestamp() int64 {
	if s.timeFunc == nil {
		return time.Now().UTC().Unix()
	}
	return s.timeFunc()
}

// Authentication -------------------------------------------------------------

// createMac creates a message authentication code (MAC).
func createMac(h hash.Hash, value []byte) []byte {
	h.Write(value)
	return h.Sum(nil)
}

// verifyMac verifies that a message authentication code (MAC) is valid.
func verifyMac(h hash.Hash, value []byte, mac []byte) error {
	mac2 := createMac(h, value)
	// Check that both MACs are of equal length, as subtle.ConstantTimeCompare
	// does not do this prior to Go 1.4.
	if len(mac) == len(mac2) && subtle.ConstantTimeCompare(mac, mac2) == 1 {
		return nil
	}
	return ErrMacInvalid
}

// Encryption -----------------------------------------------------------------

// encrypt encrypts a value using the given block in counter mode.
//
// A random initialization vector ( https://en.wikipedia.org/wiki/Block_cipher_mode_of_operation#Initialization_vector_(IV) ) with the length of the
// block size is prepended to the resulting ciphertext.
func encrypt(block cipher.Block, value []byte) ([]byte, error) {
	iv := GenerateRandomKey(block.BlockSize())
	if iv == nil {
		return nil, errGeneratingIV
	}
	// Encrypt it.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(value, value)
	// Return iv + ciphertext.
	return append(iv, value...), nil
}

// decrypt decrypts a value using the given block in counter mode.
//
// The value to be decrypted must be prepended by a initialization vector
// ( https://en.wikipedia.org/wiki/Block_cipher_mode_of_operation#Initialization_vector_(IV) ) with the length of the block size.
func decrypt(block cipher.Block, value []byte) ([]byte, error) {
	size := block.BlockSize()
	if len(value) > size {
		// Extract iv.
		iv := value[:size]
		// Extract ciphertext.
		value = value[size:]
		// Decrypt it.
		stream := cipher.NewCTR(block, iv)
		stream.XORKeyStream(value, value)
		return value, nil
	}
	return nil, errDecryptionFailed
}

// Serialization --------------------------------------------------------------

// Serialize encodes a value using gob.
func (e GobEncoder) Serialize(src interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(src); err != nil {
		return nil, cookieError{cause: err, typ: usageError}
	}
	return buf.Bytes(), nil
}

// Deserialize decodes a value using gob.
func (e GobEncoder) Deserialize(src []byte, dst interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(src))
	if err := dec.Decode(dst); err != nil {
		return cookieError{cause: err, typ: decodeError}
	}
	return nil
}

// Serialize encodes a value using encoding/json.
func (e JSONEncoder) Serialize(src interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(src); err != nil {
		return nil, cookieError{cause: err, typ: usageError}
	}
	return buf.Bytes(), nil
}

// Deserialize decodes a value using encoding/json.
func (e JSONEncoder) Deserialize(src []byte, dst interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(src))
	if err := dec.Decode(dst); err != nil {
		return cookieError{cause: err, typ: decodeError}
	}
	return nil
}

// Serialize passes a []byte through as-is.
func (e NopEncoder) Serialize(src interface{}) ([]byte, error) {
	if b, ok := src.([]byte); ok {
		return b, nil
	}

	return nil, errValueNotByte
}

// Deserialize passes a []byte through as-is.
func (e NopEncoder) Deserialize(src []byte, dst interface{}) error {
	if dat, ok := dst.(*[]byte); ok {
		*dat = src
		return nil
	}
	return errValueNotBytePtr
}

// Encoding -------------------------------------------------------------------

// encode encodes a value using base64.
func encode(value []byte) []byte {
	encoded := make([]byte, base64.URLEncoding.EncodedLen(len(value)))
	base64.URLEncoding.Encode(encoded, value)
	return encoded
}

// decode decodes a cookie using base64.
func decode(value []byte) ([]byte, error) {
	decoded := make([]byte, base64.URLEncoding.DecodedLen(len(value)))
	b, err := base64.URLEncoding.Decode(decoded, value)
	if err != nil {
		return nil, cookieError{cause: err, typ: decodeError, msg: "base64 decode failed"}
	}
	return decoded[:b], nil
}

// Helpers --------------------------------------------------------------------

// GenerateRandomKey creates a random key with the given length in bytes.
// On failure, returns nil.
//
// Note that keys created using `GenerateRandomKey()` are not automatically
// persisted. New keys will be created when the application is restarted, and
// previously issued cookies will not be able to be decoded.
//
// Callers should explicitly check for the possibility of a nil return, treat
// it as a failure of the system random number generator, and not continue.
func GenerateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}

// CodecsFromPairs returns a slice of SecureCookie instances.
//
// It is a convenience function to create a list of codecs for key rotation. Note
// that the generated Codecs will have the default options applied: callers
// should iterate over each Codec and type-assert the underlying *SecureCookie to
// change these.
//
// Example:
//
//	codecs := securecookie.CodecsFromPairs(
//	     []byte("new-hash-key"),
//	     []byte("new-block-key"),
//	     []byte("old-hash-key"),
//	     []byte("old-block-key"),
//	 )
//
//	// Modify each instance.
//	for _, s := range codecs {
//	       if cookie, ok := s.(*securecookie.SecureCookie); ok {
//	           cookie.MaxAge(86400 * 7)
//	           cookie.SetSerializer(securecookie.JSONEncoder{})
//	           cookie.HashFunc(sha512.New512_256)
//	       }
//	   }
func CodecsFromPairs(keyPairs ...[]byte) []Codec {
	codecs := make([]Codec, len(keyPairs)/2+len(keyPairs)%2)
	for i := 0; i < len(keyPairs); i += 2 {
		var blockKey []byte
		if i+1 < len(keyPairs) {
			blockKey = keyPairs[i+1]
		}
		codecs[i/2] = NewSecureCookie(keyPairs[i], blockKey)
	}
	return codecs
}

// EncodeMulti encodes a cookie value using a group of codecs.
//
// The codecs are tried in order. Multiple codecs are accepted to allow
// key rotation.
//
// On error, may return a MultiError.
func EncodeMulti(name string, value interface{}, codecs ...Codec) (string, error) {
	if len(codecs) == 0 {
		return "", errNoCodecs
	}

	var errors MultiError
	for _, codec := range codecs {
		encoded, err := codec.Encode(name, value)
		if err == nil {
			return encoded, nil
		}
		errors = append(errors, err)
	}
	return "", errors
}

// DecodeMulti decodes a cookie value using a group of codecs.
//
// The codecs are tried in order. Multiple codecs are accepted to allow
// key rotation.
//
// On error, may return a MultiError.
func DecodeMulti(name string, value string, dst interface{}, codecs ...Codec) error {
	if len(codecs) == 0 {
		return errNoCodecs
	}

	var errors MultiError
	for _, codec := range codecs {
		err := codec.Decode(name, value, dst)
		if err == nil {
			return nil
		}
		errors = append(errors, err)
	}
	return errors
}
