#include <stdio.h>
#include <stdlib.h>
#include <fcntl.h>

#define BUFSIZ 1024

int main() {
  char* buff = malloc(BUFSIZ);
  char n;
  while ((n = read(stdin, buff, BUFSIZ)) > 0) { }
  printf("{\"jsonrpc\": \"2.0\", \"result\": \"hello world\", \"id\": 0}");
  free(buff);
}
