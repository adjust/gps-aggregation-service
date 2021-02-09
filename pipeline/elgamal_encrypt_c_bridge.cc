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

#include "pipeline/elgamal_encrypt_c_bridge.h"

#include <stdio.h>

#include <cstring>
#include <iostream>
#include <string>

#include "absl/strings/string_view.h"
#include "pipeline/elgamal_encrypt.h"

using convagg::crypto::Decrypt;
using convagg::crypto::ElGamalCiphertext;
using convagg::crypto::ElGamalPrivateKey;
using convagg::crypto::ElGamalPublicKey;
using convagg::crypto::Encrypt;
using convagg::crypto::ExponentiateOnCiphertext;
using convagg::crypto::ExponentiateOnECPointStr;
using convagg::crypto::GenerateElGamalKeyPair;
using convagg::crypto::GenerateSecret;
using convagg::crypto::GetHashedECPointStrForTesting;

bool StrToCBytes(const std::string &str, CBytes *out_cb) {
  size_t size = str.size();
  // The memory will be freed in the GO code.
  char *copy = static_cast<char *>(malloc(size));
  if (copy == nullptr) {
    return false;
  }
  memcpy(copy, str.data(), size);
  out_cb->c = copy;
  out_cb->l = size;
  return true;
}

bool CGenerateElGamalKeyPair(struct CElGamalKeys *out_elgamal_keys) {
  auto maybe_key_pair = GenerateElGamalKeyPair();
  if (!maybe_key_pair.ok()) {
    return false;
  }
  return StrToCBytes(maybe_key_pair.value().public_key.g(),
                     &out_elgamal_keys->public_key.g) &&
         StrToCBytes(maybe_key_pair.value().public_key.y(),
                     &out_elgamal_keys->public_key.y) &&
         StrToCBytes(maybe_key_pair.value().private_key.x(),
                     &out_elgamal_keys->private_key.x);
}

bool CGenerateSecret(struct CBytes *out_secret_c) {
  auto maybe_secret = GenerateSecret();
  if (!maybe_secret.ok()) {
    return false;
  }
  return StrToCBytes(maybe_secret.value(), out_secret_c);
}

bool CEncrypt(const struct CBytes *message_c,
              const struct CElGamalPublicKey *public_key_c,
              struct CElGamalCiphertext *out_ciphertext_c) {
  ElGamalPublicKey public_key;
  public_key.set_g(std::string(public_key_c->g.c, public_key_c->g.l));
  public_key.set_y(std::string(public_key_c->y.c, public_key_c->y.l));

  auto maybe_ciphertext =
      Encrypt(std::string(message_c->c, message_c->l), public_key);
  if (!maybe_ciphertext.ok()) {
    return false;
  }

  return StrToCBytes(maybe_ciphertext.value().u(), &out_ciphertext_c->u) &&
         StrToCBytes(maybe_ciphertext.value().e(), &out_ciphertext_c->e);
}

bool CDecrypt(const struct CElGamalCiphertext *ciphertext_c,
              const struct CElGamalPrivateKey *private_key_c,
              struct CBytes *out_decrypted_c) {
  ElGamalCiphertext ciphertext;
  ciphertext.set_u(std::string(ciphertext_c->u.c, ciphertext_c->u.l));
  ciphertext.set_e(std::string(ciphertext_c->e.c, ciphertext_c->e.l));

  ElGamalPrivateKey private_key;
  private_key.set_x(std::string(private_key_c->x.c, private_key_c->x.l));

  auto maybe_decrypted = Decrypt(ciphertext, private_key);
  if (!maybe_decrypted.ok()) {
    return false;
  }

  return StrToCBytes(maybe_decrypted.value(), out_decrypted_c);
}

bool CExponentiateOnCiphertext(const struct CElGamalCiphertext *ciphertext_c,
                               const struct CElGamalPublicKey *public_key_c,
                               const struct CBytes *secret_exponent_c,
                               struct CElGamalCiphertext *out_result_c) {
  ElGamalCiphertext ciphertext;
  ciphertext.set_u(std::string(ciphertext_c->u.c, ciphertext_c->u.l));
  ciphertext.set_e(std::string(ciphertext_c->e.c, ciphertext_c->e.l));

  ElGamalPublicKey public_key;
  public_key.set_g(std::string(public_key_c->g.c, public_key_c->g.l));
  public_key.set_y(std::string(public_key_c->y.c, public_key_c->y.l));

  std::string secret_exponent =
      std::string(secret_exponent_c->c, secret_exponent_c->l);

  auto maybe_result =
      ExponentiateOnCiphertext(ciphertext, public_key, secret_exponent);
  if (!maybe_result.ok()) {
    return false;
  }

  return StrToCBytes(maybe_result.value().u(), &out_result_c->u) &&
         StrToCBytes(maybe_result.value().e(), &out_result_c->e);
}

bool CExponentiateOnECPointStr(const struct CBytes *value_str,
                               const struct CBytes *secret_exponent_c,
                               struct CBytes *out_result_c) {
  std::string value = std::string(value_str->c, value_str->l);
  std::string secret_exponent =
      std::string(secret_exponent_c->c, secret_exponent_c->l);
  auto maybe_result = ExponentiateOnECPointStr(value, secret_exponent);
  if (!maybe_result.ok()) {
    return false;
  }
  return StrToCBytes(maybe_result.value(), out_result_c);
}

bool CGetHashedECPointStrForTesting(const struct CBytes *message_c,
                                    struct CBytes *out_hashed_message_c) {
  std::string message = std::string(message_c->c, message_c->l);
  auto maybe_hashed_message = GetHashedECPointStrForTesting(message);
  if (!maybe_hashed_message.ok()) {
    return false;
  }
  return StrToCBytes(maybe_hashed_message.value(), out_hashed_message_c);
}
