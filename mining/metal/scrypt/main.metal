#include <metal_stdlib>

using namespace metal;

void scrypt(
    thread sha256_context *ctx,      // SHA256 context

    thread uint32_t *block,           // Input block (salt)
    size_t block_len,                  // Length of the block (salt)

    thread uint32_t *scrypt_out,       // Output buffer for the scrypt result

    device uint32_t *memBuffer
) {
    
    thread uint32_t pbkdf2_1_out[32];
    thread uint32_t romix_out[32];
    thread uint32_t pbkdf2_2_out[32];

    // First PBKDF2 operation
    pbkdf2(ctx, block, block_len, 256, pbkdf2_1_out);

    // Endian conversion
    endian_full(pbkdf2_1_out, 32);

    // Romix operation
    romix(pbkdf2_1_out, 1024, romix_out, memBuffer);

    // Endian conversion
    endian_full(romix_out, 32);

    // Second PBKDF2 operation
    pbkdf2_2nd(ctx, romix_out, 32, block, block_len, 1024, pbkdf2_2_out);

    // Copy result to output buffer
    for (uint i = 0; i < 32; i++) {
        scrypt_out[i] = pbkdf2_2_out[i];
    }
}


kernel void test_scrypt_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs,
    device uint32_t *memBuffer
) {

    const uint input_length = 32;
    const uint output_length = 32;

    thread sha256_context ctx;
    thread uint32_t thread_input[input_length];
    thread uint32_t thread_output[output_length];

    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }

    scrypt(&ctx, thread_input, 1024, thread_output, memBuffer);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}