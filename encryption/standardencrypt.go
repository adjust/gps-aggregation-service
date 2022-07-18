// Copyright 2021 Google LLC
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

// Package standardencrypt contains functions for the standard public-key encryption.
package standardencrypt

import (
	"bytes"
	"errors"

	"github.com/google/tink/go/hybrid"
	"github.com/google/tink/go/hybrid/subtle"
	"github.com/google/tink/go/insecurecleartextkeyset"
	"github.com/google/tink/go/keyset"

	pb "github.com/google/privacy-sandbox-aggregation-service/encryption/crypto_go_proto"
	tpb "github.com/google/tink/go/proto/tink_go_proto"
)

// KeyTemplate specifies the parameters for HPKE encryption.
func KeyTemplate() *tpb.KeyTemplate {
	return hybrid.DHKEM_X25519_HKDF_SHA256_HKDF_SHA256_CHACHA20_POLY1305_Raw_Key_Template()
}

// GenerateStandardKeyPair generates a private key and a corresponding public key.
func GenerateStandardKeyPair() (*pb.StandardPrivateKey, *pb.StandardPublicKey, error) {
	priv, err := keyset.NewHandle(KeyTemplate())
	if err != nil {
		return nil, nil, err
	}
	bPriv := new(bytes.Buffer)
	err = insecurecleartextkeyset.Write(priv, keyset.NewBinaryWriter(bPriv))
	if err != nil {
		return nil, nil, err
	}
	privateKey := &pb.StandardPrivateKey{Key: bPriv.Bytes()}

	pub, err := priv.Public()
	if err != nil {
		return nil, nil, err
	}
	bPub, err := subtle.SerializePrimaryPublicKey(pub, KeyTemplate())
	if err != nil {
		return nil, nil, err
	}
	publicKey := &pb.StandardPublicKey{Key: bPub}

	return privateKey, publicKey, nil
}

// Encrypt encrypts the input message with the given public key.
func Encrypt(message, context []byte, publicKey *pb.StandardPublicKey) (*pb.StandardCiphertext, error) {
	pub, err := subtle.KeysetHandleFromSerializedPublicKey(publicKey.Key, KeyTemplate())
	if err != nil {
		return nil, err
	}

	he, err := hybrid.NewHybridEncrypt(pub)
	if err != nil {
		return nil, err
	}
	ct, err := he.Encrypt(message, context)
	if err != nil {
		return nil, err
	}
	return &pb.StandardCiphertext{Data: ct}, err
}

// Decrypt decrypts the message with the given private key.
func Decrypt(encrypted *pb.StandardCiphertext, context []byte, privateKey *pb.StandardPrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, errors.New("empty private key")
	}
	bPriv := bytes.NewBuffer(privateKey.Key)
	priv, err := insecurecleartextkeyset.Read(keyset.NewBinaryReader(bPriv))
	if err != nil {
		return nil, err
	}

	hd, err := hybrid.NewHybridDecrypt(priv)
	if err != nil {
		return nil, err
	}
	return hd.Decrypt(encrypted.Data, context)
}
