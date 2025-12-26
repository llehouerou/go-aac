# FAAD2 Debug Tool Makefile
#
# This builds tools for generating reference data from FAAD2
# to test the go-aac implementation.

FAAD2_DIR := $(HOME)/dev/faad2
FAAD2_LIB := $(FAAD2_DIR)/libfaad
FAAD2_INC := $(FAAD2_DIR)/include

# Check if FAAD2 exists
ifeq ($(wildcard $(FAAD2_DIR)/libfaad/decoder.c),)
$(error FAAD2 not found at $(FAAD2_DIR). Clone it with: git clone https://github.com/knik0/faad2 ~/dev/faad2)
endif

CC := gcc
CFLAGS := -O2 -Wall -Wextra -I$(FAAD2_INC) -I$(FAAD2_DIR)/libfaad

# FAAD2 source files needed for the debug tool
FAAD2_SOURCES := \
	$(FAAD2_LIB)/bits.c \
	$(FAAD2_LIB)/cfft.c \
	$(FAAD2_LIB)/common.c \
	$(FAAD2_LIB)/decoder.c \
	$(FAAD2_LIB)/drc.c \
	$(FAAD2_LIB)/drm_dec.c \
	$(FAAD2_LIB)/error.c \
	$(FAAD2_LIB)/filtbank.c \
	$(FAAD2_LIB)/hcr.c \
	$(FAAD2_LIB)/huffman.c \
	$(FAAD2_LIB)/ic_predict.c \
	$(FAAD2_LIB)/is.c \
	$(FAAD2_LIB)/lt_predict.c \
	$(FAAD2_LIB)/mdct.c \
	$(FAAD2_LIB)/mp4.c \
	$(FAAD2_LIB)/ms.c \
	$(FAAD2_LIB)/output.c \
	$(FAAD2_LIB)/pns.c \
	$(FAAD2_LIB)/ps_dec.c \
	$(FAAD2_LIB)/ps_syntax.c \
	$(FAAD2_LIB)/pulse.c \
	$(FAAD2_LIB)/rvlc.c \
	$(FAAD2_LIB)/sbr_dct.c \
	$(FAAD2_LIB)/sbr_dec.c \
	$(FAAD2_LIB)/sbr_e_nf.c \
	$(FAAD2_LIB)/sbr_fbt.c \
	$(FAAD2_LIB)/sbr_hfadj.c \
	$(FAAD2_LIB)/sbr_hfgen.c \
	$(FAAD2_LIB)/sbr_huff.c \
	$(FAAD2_LIB)/sbr_qmf.c \
	$(FAAD2_LIB)/sbr_syntax.c \
	$(FAAD2_LIB)/sbr_tf_grid.c \
	$(FAAD2_LIB)/specrec.c \
	$(FAAD2_LIB)/ssr.c \
	$(FAAD2_LIB)/ssr_fb.c \
	$(FAAD2_LIB)/ssr_ipqf.c \
	$(FAAD2_LIB)/syntax.c \
	$(FAAD2_LIB)/tns.c

# Define PACKAGE_VERSION (required by FAAD2)
FAAD2_VERSION := $(shell grep -o '"[0-9]*\.[0-9]*\.[0-9]*"' $(FAAD2_DIR)/properties.json 2>/dev/null | tr -d '"')
ifeq ($(FAAD2_VERSION),)
FAAD2_VERSION := 2.11.1
endif

FAAD2_DEFINES := \
	-DPACKAGE_VERSION=\"$(FAAD2_VERSION)\" \
	-DHAVE_INTTYPES_H=1 \
	-DHAVE_MEMCPY=1 \
	-DHAVE_STDINT_H=1 \
	-DHAVE_STRING_H=1 \
	-DHAVE_STRINGS_H=1 \
	-DHAVE_STDLIB_H=1

.PHONY: all clean test help install

all: faad2_debug

help:
	@echo "FAAD2 Debug Tools for go-aac"
	@echo ""
	@echo "Targets:"
	@echo "  all       - Build faad2_debug tool"
	@echo "  test      - Run a quick test"
	@echo "  clean     - Remove built files"
	@echo "  install   - Copy faad2_debug to /tmp for easy access"
	@echo ""
	@echo "Usage:"
	@echo "  ./faad2_debug <input.aac> <output_dir> [max_frames]"
	@echo ""
	@echo "FAAD2 location: $(FAAD2_DIR)"
	@echo "FAAD2 version: $(FAAD2_VERSION)"

# Basic debug tool using public API
faad2_debug: faad2_debug.c $(FAAD2_SOURCES)
	$(CC) $(CFLAGS) $(FAAD2_DEFINES) -o $@ $^ -lm
	@echo "Built faad2_debug successfully"

clean:
	rm -f faad2_debug
	rm -rf test_output

install: faad2_debug
	cp faad2_debug /tmp/faad2_debug
	chmod +x /tmp/faad2_debug
	@echo "Installed to /tmp/faad2_debug"

# Quick test with a generated AAC file
test: faad2_debug
	@mkdir -p test_output
	@echo "Generating test AAC file..."
	@ffmpeg -y -f lavfi -i "sine=frequency=1000:duration=0.1" -c:a aac -b:a 128k test_output/test.aac 2>/dev/null || \
		(echo "Error: ffmpeg not found. Install ffmpeg to run tests." && exit 1)
	@echo ""
	@echo "Running faad2_debug..."
	./faad2_debug test_output/test.aac test_output 5
	@echo ""
	@echo "Output files:"
	@ls -la test_output/
	@echo ""
	@cat test_output/info.json
	@echo ""
	@echo "Test complete!"
