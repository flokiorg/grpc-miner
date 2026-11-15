#include <metal_stdlib>

using namespace metal;



#define ROTLEFT(x, b) (((x) << (b)) | ((x) >> (32 - (b))))
#define SUM(a1, a2) ((a1) + (a2))
#define SALSA_MIX(destination, a1, a2, b) ((destination) ^ (ROTLEFT(SUM((a1), (a2)), (b))))


void salsa_round(thread uint32_t *x1, thread uint32_t *x2, thread uint32_t *x3, thread uint32_t *x4) {
    *x1 = SALSA_MIX(*x1, *x4, *x3, 7);
    *x2 = SALSA_MIX(*x2, *x1, *x4, 9);
    *x3 = SALSA_MIX(*x3, *x2, *x1, 13);
    *x4 = SALSA_MIX(*x4, *x3, *x2, 18);
}

void salsa20_8(thread uint32_t *x, thread uint32_t *out) {
    for (int i = 0; i < 4; i++) {
        salsa_round(&x[4], &x[8], &x[12], &x[0]);
        salsa_round(&x[9], &x[13], &x[1], &x[5]);
        salsa_round(&x[14], &x[2], &x[6], &x[10]);
        salsa_round(&x[3], &x[7], &x[11], &x[15]);
        salsa_round(&x[1], &x[2], &x[3], &x[0]);
        salsa_round(&x[6], &x[7], &x[4], &x[5]);
        salsa_round(&x[11], &x[8], &x[9], &x[10]);
        salsa_round(&x[12], &x[13], &x[14], &x[15]);
    }
    for (int i = 0; i < 16; i++) {
        out[i] = x[i];
    }
}


void add_two_uint32_ts_array_512_bit(thread uint32_t *a, thread uint32_t *b) {
    // Add two 512-bit (16 uint32_ts) arrays element-wise
    for (int i = 15; i >= 0; i--) {
        a[i] += b[i];
    }
}

void blockmix(thread uint32_t *block, thread uint32_t *out) {
    uint32_t x_arr[16];
    uint32_t x_arr_cpy[16];

    // Copy the first 16 elements of the block
    for (int i = 0; i < 16; i++) {
        x_arr[i] = block[i];
    }

    for (int i = 0; i < 2; i++) {
        for (int j = 0; j < 16; j++) {
            x_arr_cpy[j] = x_arr[j] ^ block[j + 16];
            x_arr[j] ^= block[j + 16];
        }

        uint32_t salsa_out[16];
        salsa20_8(x_arr_cpy, salsa_out);
        add_two_uint32_ts_array_512_bit(x_arr, salsa_out);

        // Store the result in the output array
        for (int j = 0; j < 16; j++) {
            out[(16 * i) + j] = x_arr[j];
        }
    }
}


void romix_old(
    thread uint32_t *block,  // Input and output block buffer
    size_t N,                  // Number of iterations
    thread uint32_t *out
) {
    uint32_t mem[1024][32];
    uint j;

    // First phase: fill memory with block states
    for (uint i = 0; i < N; i++) {
        for (j = 0; j < 32; j++) {
            mem[i][j] = block[j];
        }
        blockmix(block, block);  // Assuming blockmix is available and modifies block in place
    }

    // Second phase: XOR block with memory and apply blockmix
    for (uint i = 0; i < N; i++) {
        j = (block[16] & 0x000003ff);  // Extract index within memory bounds

        for (uint k = 0; k < 32; k++) {
            block[k] ^= mem[j][k];
        }
        blockmix(block, block);  // Reapply blockmix on the updated block
    }

    // Copy result to output buffer
    for (uint i = 0; i < 32; i++) {
        out[i] = block[i];
    }
}

void romix(
    thread uint32_t *block,              // Input and output block buffer
    size_t N,                            // Number of iterations
    thread uint32_t *out,                 // Output block buffer
    device uint32_t *memBuffer          // Device buffer for memory
) {
    uint j;

    // First phase: fill memory with block states
    for (uint i = 0; i < N; i++) {
        for (uint k = 0; k < 32; k++) {
            memBuffer[i * 32 + k] = block[k];  // Store block state in device memory
        }
        blockmix(block, block);  // Assuming blockmix is available and modifies block in place
    }

    // Second phase: XOR block with memory and apply blockmix
    for (uint i = 0; i < N; i++) {
        j = (block[16] & 0x000003ff);  // Extract index within memory bounds

        for (uint k = 0; k < 32; k++) {
            block[k] ^= memBuffer[j * 32 + k];  // Access device memory
        }
        blockmix(block, block);  // Reapply blockmix on the updated block
    }

    // Copy result to output buffer
    for (uint i = 0; i < 32; i++) {
        out[i] = block[i];
    }
}

void endian_full(thread uint32_t *buffer, size_t length) {

    for (size_t i = 0; i < length; i++) {
        uint32_t w = buffer[i];
        buffer[i] = (w>>24)|((w>>8)&0x0000ff00)|((w<<8)&0x00ff0000)|(w<<24);
    }
}


////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////



kernel void test_romix_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs,
    device uint32_t *memBuffer
) {

    const uint input_length = 32;
    const uint output_length = 32;

    thread uint32_t thread_input[input_length];
    thread uint32_t thread_output[output_length];

    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }

    romix(thread_input, 1024, thread_output, memBuffer);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}

kernel void test_salsa_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 16;
    const uint output_length = 16;

    thread uint32_t thread_input[input_length];
    thread uint32_t thread_output[output_length];

    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }

    salsa20_8(thread_input, thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}



kernel void test_blockmix_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 32;
    const uint output_length = 32;

    thread uint32_t thread_input[input_length];
    thread uint32_t thread_output[output_length];

    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }

    blockmix(thread_input, thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}




kernel void test_endian_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint length = 32;

    thread uint32_t thread_input[length];

    for (uint i = 0; i < length; i++) {
        thread_input[i] = inputs[i];
    }

    endian_full(thread_input, length);

    for (uint i = 0; i < length; i++) {
        outputs[i] = thread_input[i];
    }
}



