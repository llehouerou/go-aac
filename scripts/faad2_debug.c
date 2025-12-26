/*
 * FAAD2 Debug Tool for go-aac
 *
 * This tool decodes AAC files using FAAD2 and dumps intermediate values
 * at each stage of the decode pipeline for comparison with the Go implementation.
 *
 * Build: See Makefile in this directory
 * Usage: ./faad2_debug <input.aac> <output_dir> [frame_number]
 *
 * Output files (binary, little-endian):
 *   frame_N_adts.bin      - ADTS header fields (if ADTS)
 *   frame_N_spec_int.bin  - Huffman-decoded spectral coefficients (int16[1024])
 *   frame_N_spec_float.bin - After dequant+scalefactors (float32[1024])
 *   frame_N_spec_tns.bin  - After TNS (float32[1024])
 *   frame_N_time.bin      - After IMDCT (float32[1024 or 2048])
 *   frame_N_pcm.bin       - Final PCM output (int16[samples])
 *   info.json             - Decode info (sample rate, channels, etc.)
 */

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

/* FAAD2 public API */
#include "neaacdec.h"

/* We need access to internal structures for dumping intermediate values.
 * This requires compiling with FAAD2 source or a modified library.
 * For now, we'll use the public API and dump what we can access.
 */

#define MAX_FRAME_SIZE (768 * 8)  /* Max AAC frame size */
#define ADTS_HEADER_SIZE 7

typedef struct {
    uint16_t syncword;
    uint8_t  id;
    uint8_t  layer;
    uint8_t  protection_absent;
    uint8_t  profile;
    uint8_t  sf_index;
    uint8_t  private_bit;
    uint8_t  channel_config;
    uint8_t  original;
    uint8_t  home;
    uint8_t  copyright_bit;
    uint8_t  copyright_start;
    uint16_t frame_length;
    uint16_t buffer_fullness;
    uint8_t  num_raw_blocks;
} adts_header_t;

/* Parse ADTS header manually for dumping */
static int parse_adts_header(const uint8_t *data, size_t len, adts_header_t *hdr) {
    if (len < ADTS_HEADER_SIZE) return -1;

    /* Check sync word */
    if (data[0] != 0xFF || (data[1] & 0xF0) != 0xF0) {
        return -1;
    }

    hdr->syncword = 0xFFF;
    hdr->id = (data[1] >> 3) & 0x01;
    hdr->layer = (data[1] >> 1) & 0x03;
    hdr->protection_absent = data[1] & 0x01;
    hdr->profile = (data[2] >> 6) & 0x03;
    hdr->sf_index = (data[2] >> 2) & 0x0F;
    hdr->private_bit = (data[2] >> 1) & 0x01;
    hdr->channel_config = ((data[2] & 0x01) << 2) | ((data[3] >> 6) & 0x03);
    hdr->original = (data[3] >> 5) & 0x01;
    hdr->home = (data[3] >> 4) & 0x01;
    hdr->copyright_bit = (data[3] >> 3) & 0x01;
    hdr->copyright_start = (data[3] >> 2) & 0x01;
    hdr->frame_length = ((data[3] & 0x03) << 11) | (data[4] << 3) | ((data[5] >> 5) & 0x07);
    hdr->buffer_fullness = ((data[5] & 0x1F) << 6) | ((data[6] >> 2) & 0x3F);
    hdr->num_raw_blocks = data[6] & 0x03;

    return 0;
}

/* Sample rate table */
static const uint32_t sample_rates[] = {
    96000, 88200, 64000, 48000, 44100, 32000,
    24000, 22050, 16000, 12000, 11025, 8000, 7350,
    0, 0, 0
};

/* Find next ADTS sync word */
static int find_adts_sync(const uint8_t *data, size_t len) {
    for (size_t i = 0; i + 1 < len; i++) {
        if (data[i] == 0xFF && (data[i+1] & 0xF0) == 0xF0) {
            return (int)i;
        }
    }
    return -1;
}

/* Dump binary data to file */
static int dump_binary(const char *path, const void *data, size_t size) {
    FILE *f = fopen(path, "wb");
    if (!f) {
        fprintf(stderr, "Error: cannot create %s\n", path);
        return -1;
    }
    fwrite(data, 1, size, f);
    fclose(f);
    return 0;
}

/* Dump ADTS header to binary file */
static int dump_adts_header(const char *dir, int frame, const adts_header_t *hdr) {
    char path[512];
    snprintf(path, sizeof(path), "%s/frame_%04d_adts.bin", dir, frame);

    /* Write as packed struct for easy parsing */
    uint8_t buf[16];
    buf[0] = (hdr->syncword >> 8) & 0xFF;
    buf[1] = hdr->syncword & 0xFF;
    buf[2] = hdr->id;
    buf[3] = hdr->layer;
    buf[4] = hdr->protection_absent;
    buf[5] = hdr->profile;
    buf[6] = hdr->sf_index;
    buf[7] = hdr->private_bit;
    buf[8] = hdr->channel_config;
    buf[9] = hdr->original;
    buf[10] = hdr->home;
    buf[11] = (hdr->frame_length >> 8) & 0xFF;
    buf[12] = hdr->frame_length & 0xFF;
    buf[13] = (hdr->buffer_fullness >> 8) & 0xFF;
    buf[14] = hdr->buffer_fullness & 0xFF;
    buf[15] = hdr->num_raw_blocks;

    return dump_binary(path, buf, 16);
}

/* Dump PCM samples */
static int dump_pcm(const char *dir, int frame, const void *samples,
                    size_t num_samples, int channels, int format) {
    char path[512];
    snprintf(path, sizeof(path), "%s/frame_%04d_pcm.bin", dir, frame);

    size_t sample_size;
    switch (format) {
        case FAAD_FMT_16BIT: sample_size = 2; break;
        case FAAD_FMT_24BIT: sample_size = 4; break; /* stored as int32 */
        case FAAD_FMT_32BIT: sample_size = 4; break;
        case FAAD_FMT_FLOAT: sample_size = 4; break;
        case FAAD_FMT_DOUBLE: sample_size = 8; break;
        default: sample_size = 2; break;
    }

    return dump_binary(path, samples, num_samples * channels * sample_size);
}

/* Write info.json */
static int write_info_json(const char *dir, unsigned long sample_rate,
                           unsigned char channels, int total_frames,
                           unsigned long total_samples) {
    char path[512];
    snprintf(path, sizeof(path), "%s/info.json", dir);

    FILE *f = fopen(path, "w");
    if (!f) return -1;

    fprintf(f, "{\n");
    fprintf(f, "  \"sample_rate\": %lu,\n", sample_rate);
    fprintf(f, "  \"channels\": %d,\n", channels);
    fprintf(f, "  \"total_frames\": %d,\n", total_frames);
    fprintf(f, "  \"total_samples\": %lu,\n", total_samples);
    fprintf(f, "  \"format\": \"int16\"\n");
    fprintf(f, "}\n");

    fclose(f);
    return 0;
}

static void usage(const char *prog) {
    fprintf(stderr, "Usage: %s <input.aac> <output_dir> [max_frames]\n", prog);
    fprintf(stderr, "\n");
    fprintf(stderr, "Decodes AAC file and dumps intermediate values for testing.\n");
    fprintf(stderr, "\n");
    fprintf(stderr, "Arguments:\n");
    fprintf(stderr, "  input.aac   - Input AAC file (ADTS format)\n");
    fprintf(stderr, "  output_dir  - Directory for output files\n");
    fprintf(stderr, "  max_frames  - Maximum frames to decode (default: all)\n");
    fprintf(stderr, "\n");
    fprintf(stderr, "Output files per frame:\n");
    fprintf(stderr, "  frame_NNNN_adts.bin - ADTS header (16 bytes)\n");
    fprintf(stderr, "  frame_NNNN_pcm.bin  - PCM samples (int16, interleaved)\n");
    fprintf(stderr, "  info.json           - Decode metadata\n");
}

int main(int argc, char *argv[]) {
    if (argc < 3) {
        usage(argv[0]);
        return 1;
    }

    const char *input_path = argv[1];
    const char *output_dir = argv[2];
    int max_frames = argc > 3 ? atoi(argv[3]) : INT32_MAX;

    /* Read input file */
    FILE *f = fopen(input_path, "rb");
    if (!f) {
        fprintf(stderr, "Error: cannot open %s\n", input_path);
        return 1;
    }

    fseek(f, 0, SEEK_END);
    size_t file_size = ftell(f);
    fseek(f, 0, SEEK_SET);

    uint8_t *file_data = malloc(file_size);
    if (!file_data) {
        fprintf(stderr, "Error: out of memory\n");
        fclose(f);
        return 1;
    }

    if (fread(file_data, 1, file_size, f) != file_size) {
        fprintf(stderr, "Error: cannot read file\n");
        free(file_data);
        fclose(f);
        return 1;
    }
    fclose(f);

    /* Check for ADTS sync */
    int sync_offset = find_adts_sync(file_data, file_size);
    if (sync_offset < 0) {
        fprintf(stderr, "Error: not an ADTS file (no sync word found)\n");
        free(file_data);
        return 1;
    }
    if (sync_offset > 0) {
        fprintf(stderr, "Warning: skipped %d bytes to find ADTS sync\n", sync_offset);
    }

    /* Initialize FAAD2 decoder */
    NeAACDecHandle decoder = NeAACDecOpen();
    if (!decoder) {
        fprintf(stderr, "Error: cannot create decoder\n");
        free(file_data);
        return 1;
    }

    /* Configure decoder */
    NeAACDecConfigurationPtr config = NeAACDecGetCurrentConfiguration(decoder);
    config->outputFormat = FAAD_FMT_16BIT;
    config->downMatrix = 0;
    NeAACDecSetConfiguration(decoder, config);

    /* Initialize with first frame */
    unsigned long sample_rate = 0;
    unsigned char channels = 0;

    long bytes_consumed = NeAACDecInit(decoder, file_data + sync_offset,
                                        file_size - sync_offset,
                                        &sample_rate, &channels);
    if (bytes_consumed < 0) {
        fprintf(stderr, "Error: decoder init failed\n");
        NeAACDecClose(decoder);
        free(file_data);
        return 1;
    }

    printf("Initialized: %lu Hz, %d channels\n", sample_rate, channels);
    printf("Output directory: %s\n", output_dir);

    /* Decode frames */
    size_t pos = sync_offset;
    int frame_num = 0;
    unsigned long total_samples = 0;

    while (pos < file_size && frame_num < max_frames) {
        /* Find next ADTS frame */
        int next_sync = find_adts_sync(file_data + pos, file_size - pos);
        if (next_sync < 0) break;
        pos += next_sync;

        /* Parse ADTS header for dumping */
        adts_header_t adts_hdr;
        if (parse_adts_header(file_data + pos, file_size - pos, &adts_hdr) < 0) {
            fprintf(stderr, "Warning: invalid ADTS header at frame %d\n", frame_num);
            pos++;
            continue;
        }

        /* Validate frame length */
        if (adts_hdr.frame_length < ADTS_HEADER_SIZE ||
            pos + adts_hdr.frame_length > file_size) {
            fprintf(stderr, "Warning: invalid frame length at frame %d\n", frame_num);
            pos++;
            continue;
        }

        /* Dump ADTS header */
        dump_adts_header(output_dir, frame_num, &adts_hdr);

        /* Decode frame */
        NeAACDecFrameInfo frame_info;
        void *samples = NeAACDecDecode(decoder, &frame_info,
                                       file_data + pos,
                                       file_size - pos);

        if (frame_info.error) {
            fprintf(stderr, "Warning: decode error at frame %d: %s\n",
                    frame_num, NeAACDecGetErrorMessage(frame_info.error));
            pos += adts_hdr.frame_length;
            frame_num++;
            continue;
        }

        if (samples && frame_info.samples > 0) {
            /* Dump PCM output */
            dump_pcm(output_dir, frame_num, samples,
                     frame_info.samples / frame_info.channels,
                     frame_info.channels, FAAD_FMT_16BIT);
            total_samples += frame_info.samples / frame_info.channels;
        }

        printf("Frame %d: %lu samples, %d channels\n",
               frame_num, frame_info.samples, frame_info.channels);

        pos += adts_hdr.frame_length;
        frame_num++;
    }

    /* Write info.json */
    write_info_json(output_dir, sample_rate, channels, frame_num, total_samples);

    printf("\nDecoded %d frames, %lu total samples\n", frame_num, total_samples);

    NeAACDecClose(decoder);
    free(file_data);

    return 0;
}
