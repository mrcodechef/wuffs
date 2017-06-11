#ifndef PUFFS_GIF_H
#define PUFFS_GIF_H

// Code generated by puffs-gen-c. DO NOT EDIT.

#ifndef PUFFS_BASE_HEADER_H
#define PUFFS_BASE_HEADER_H

// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

#include <stdbool.h>
#include <stdint.h>
#include <string.h>

// Puffs requires a word size of at least 32 bits because it assumes that
// converting a u32 to usize will never overflow. For example, the size of a
// decoded image is often represented, explicitly or implicitly in an image
// file, as a u32, and it is convenient to compare that to a buffer size.
#if __WORDSIZE < 32
#error "Puffs requires a word size of at least 32 bits"
#endif

// PUFFS_VERSION is the major.minor version number as a uint32. The major
// number is the high 16 bits. The minor number is the low 16 bits.
//
// The intention is to bump the version number at least on every API / ABI
// backwards incompatible change.
//
// For now, the API and ABI are simply unstable and can change at any time.
//
// TODO: don't hard code this in base-header.h.
#define PUFFS_VERSION (0x00001)

// puffs_base_buf1 is a 1-dimensional buffer (a pointer and length) plus
// additional indexes into that buffer.
//
// A value with all fields NULL or zero is a valid, empty buffer.
typedef struct {
  uint8_t* ptr;  // Pointer.
  size_t len;    // Length.
  size_t wi;     // Write index. Invariant: wi <= len.
  size_t ri;     // Read  index. Invariant: ri <= wi.
} puffs_base_buf1;

#endif  // PUFFS_BASE_HEADER_H

#ifdef __cplusplus
extern "C" {
#endif

// ---------------- Status Codes

// Status codes are non-positive integers.
//
// The least significant bit indicates a non-recoverable status code: an error.
typedef enum {
  puffs_gif_status_ok = 0,
  puffs_gif_error_bad_version = -2 + 1,
  puffs_gif_error_bad_receiver = -4 + 1,
  puffs_gif_error_bad_argument = -6 + 1,
  puffs_gif_error_constructor_not_called = -8 + 1,
  puffs_gif_error_unexpected_eof = -10 + 1,
  puffs_gif_status_short_read = -12,
  puffs_gif_status_short_write = -14,
  puffs_gif_error_lzw_code_is_out_of_range = -256 + 1,
  puffs_gif_error_lzw_prefix_chain_is_cyclical = -258 + 1,
} puffs_gif_status;

bool puffs_gif_status_is_error(puffs_gif_status s);

const char* puffs_gif_status_string(puffs_gif_status s);

// ---------------- Public Structs

typedef struct {
  // Do not access the private_impl's fields directly. There is no API/ABI
  // compatibility or safety guarantee if you do so. Instead, use the
  // puffs_gif_lzw_decoder_etc functions.
  //
  // In C++, these fields would be "private", but C does not support that.
  //
  // It is a struct, not a struct*, so that it can be stack allocated.
  struct {
    puffs_gif_status status;
    uint32_t magic;
    uint32_t f_literal_width;
    uint8_t f_stack[4096];
    uint8_t f_suffixes[4096];
    uint16_t f_prefixes[4096];
  } private_impl;
} puffs_gif_lzw_decoder;

// ---------------- Public Constructor and Destructor Prototypes

// puffs_gif_lzw_decoder_constructor is a constructor function.
//
// It should be called before any other puffs_gif_lzw_decoder_* function.
//
// Pass PUFFS_VERSION and 0 for puffs_version and for_internal_use_only.
void puffs_gif_lzw_decoder_constructor(puffs_gif_lzw_decoder* self,
                                       uint32_t puffs_version,
                                       uint32_t for_internal_use_only);

void puffs_gif_lzw_decoder_destructor(puffs_gif_lzw_decoder* self);

// ---------------- Public Function Prototypes

void puffs_gif_lzw_decoder_set_literal_width(puffs_gif_lzw_decoder* self,
                                             uint32_t a_lw);

puffs_gif_status puffs_gif_lzw_decoder_decode(puffs_gif_lzw_decoder* self,
                                              puffs_base_buf1* a_dst,
                                              puffs_base_buf1* a_src,
                                              bool a_src_final);

#ifdef __cplusplus
}  // extern "C"
#endif

#endif  // PUFFS_GIF_H
