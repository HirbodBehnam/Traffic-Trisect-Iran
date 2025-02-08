#include <errno.h>
#include <stdio.h>
#include <stdlib.h>

#define BUFFER_SIZE 32 * 1024
#define MAX_FILENAME 256

int main(int argc, char **argv) {
    // check arguments
    if (argc < 2) {
        puts("Please pass the file name without the .1 extension as argument to program.");
        exit(2);
    }
    // open destination file
    FILE *dst = fopen(argv[1], "wb");
    if (dst == NULL) {
        perror("Cannot open the destination file");
        exit(1);
    }
    // start reading from the first file
    int counter = 1; // count the extension
    while (1) {
        // create the filename and read it
        char filename[MAX_FILENAME];
        sprintf(filename, "%s.%d", argv[1], counter); // a buffer overflow might occur here
        FILE *src = fopen(filename, "rb");
        if (src == NULL) // terminate the loop if the file does not exists
            break;
        // report progress
        printf("\rProcessing file %d", counter);
        // read the file
        unsigned char buffer[BUFFER_SIZE];
        while (!feof(src)) {
            size_t read_bytes = fread(buffer, 1, sizeof(buffer), src); // read from file
            fwrite(buffer, 1, read_bytes, dst); // write to file
        }
        fclose(src);
        // increase the counter
        counter++;
    }
	putchar('\n');
    // close dst
    fclose(dst);
    return 0;
}
