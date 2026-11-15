#include <metal_stdlib>

using namespace metal;

void pbkdf2(
    thread sha256_context *ctx,    // SHA256 context

    thread uint32_t *block,         // Input block (salt)
    const size_t block_len,                // Length of the block (salt)

    size_t dklenP,                   // Desired key length
    thread uint32_t *pbkdf2_out     // Output buffer for the PBKDF2 result
) {
    int num_loop = 1024 / dklenP;
    uint32_t salt[21];
    uint32_t hmac_out[8];  // Buffer for HMAC result

    // Copy block (salt) into the salt array
    for (uint i = 0; i < block_len; i++) {
        salt[i] = block[i];
    }

    // Perform the PBKDF2 algorithm
    for (uint8_t i = 1; i <= num_loop; i++) {
        salt[block_len] = i;  // Append loop counter to the salt

        // Compute HMAC using the current salt and block
        hmac(ctx, salt, block_len + 1, block, block_len, hmac_out);

        // Store the result of HMAC in the output array
        for (uint j = 0; j < 8; j++) {
            pbkdf2_out[(i - 1) * 8 + j] = hmac_out[j];
        }
    }
}

void pbkdf2_2nd(
    thread sha256_context *ctx,    // SHA256 context

    thread uint32_t *rm_out,         // Input block (salt)
    const size_t rm_out_len,                // Length of the block (salt)

    thread uint32_t *block,         // Input block (salt)
    const size_t block_len,                // Length of the block (salt)

    size_t dklenP,                   // Desired key length
    thread uint32_t *pbkdf2_out     // Output buffer for the PBKDF2 result
) {
    int num_loop = 1024 / dklenP;
    uint32_t salt[33];
    uint32_t hmac_out[8];  // Buffer for HMAC result

    // Copy block (salt) into the salt array
    for (uint i = 0; i < rm_out_len; i++) {
        salt[i] = rm_out[i];
    }

    // Perform the PBKDF2 algorithm
    for (uint8_t i = 1; i <= num_loop; i++) {
        salt[rm_out_len] = i;  // Append loop counter to the salt

        // Compute HMAC using the current salt and block
        hmac(ctx, salt, rm_out_len + 1, block, block_len, hmac_out);

        // Store the result of HMAC in the output array
        for (uint j = 0; j < 8; j++) {
            pbkdf2_out[(i - 1) * 8 + j] = hmac_out[j];
        }
    }
}


kernel void test_pbkdf2_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 20;
    const uint output_length = 32;

    thread sha256_context ctx;
    thread uint32_t thread_input[input_length];
    thread uint32_t thread_output[output_length];

    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }

    pbkdf2(&ctx, thread_input, input_length, 256, thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}