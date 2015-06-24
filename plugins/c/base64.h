#ifndef __BASE64_H
#define __BASE64_H

#define BASE64e(msg, outlen) base64_encode((unsigned char*)(msg), strlen(msg), outlen)
#define BASE64d(msg, outlen) (char*)base64_decode(msg, strlen(msg), outlen)

void build_decoding_table();
void base64_cleanup();
char *base64_encode(const unsigned char *data, size_t input_length, size_t *output_length);
unsigned char *base64_decode(const char *data, size_t input_length, size_t *output_length);

#endif
