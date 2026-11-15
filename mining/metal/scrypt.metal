#include <metal_stdlib>

using namespace metal;






#define MAX_BLOCK_LEN 64
#define MAX_RM_OUT_LEN 64 
#define MAX_PBKDF2_OUT_LEN 8
#define MAX_N 1024

void endian_full(thread uint8_t *buffer, size_t length) {
    // Ensure length is a multiple of 4 for 32-bit word reversal
    if (length % 4 != 0) {
        return; // Exit if length is not compatible with 32-bit word size
    }
    
    for (size_t i = 0; i < length; i += 4) {
        // Swap the bytes within each 4-byte word
        uint8_t temp = buffer[i];
        buffer[i] = buffer[i + 3];
        buffer[i + 3] = temp;
        
        temp = buffer[i + 1];
        buffer[i + 1] = buffer[i + 2];
        buffer[i + 2] = temp;
    }
}





void romix(
    uint8_t block[MAX_PBKDF2_OUT_LEN],  // Input and output block buffer
    uint N,                  // Number of iterations
    uint8_t out[32]
) {
    uint8_t mem[MAX_N][32];
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


void pbkdf2_2nd(
    thread sha256_context *ctx,    // SHA256 context, same order as in C++
    thread uint8_t *rm_out,        // First round PBKDF2 output (rm_out)
    size_t rm_out_len,               // Length of rm_out
    thread uint8_t *block,         // Input block (salt)
    size_t block_len,                // Length of the block (salt)
    size_t dklenP,                   // Desired key length
    thread uint8_t *pbkdf2_out     // Output buffer for the PBKDF2 result
) {
    size_t num_loop = 1024 / dklenP;
    uint8_t salt[MAX_RM_OUT_LEN + 1];
    uint8_t hmac_out[32];  // Buffer for the HMAC result

    // Copy rm_out into the salt array
    for (uint i = 0; i < rm_out_len; i++) {
        salt[i] = rm_out[i];
    }

    // Perform the PBKDF2 algorithm
    for (uint i = 1; i <= num_loop; i++) {
        salt[rm_out_len] = static_cast<uint8_t>(i);  // Append loop counter to salt

        // Compute HMAC using the current salt and block
        hmac(ctx, salt, rm_out_len + 1, block, block_len, hmac_out);

        // Store the result of HMAC in the output array
        for (uint j = 0; j < 8; j++) {
            pbkdf2_out[(i - 1) * 8 + j] = hmac_out[j];
        }
    }
}


void scrypt(
    thread sha256_context *ctx,      // SHA256 context

    thread uint8_t *block,           // Input block (salt)
    size_t block_len,                  // Length of the block (salt)

    thread uint8_t *scrypt_out       // Output buffer for the scrypt result
) {

    uint N = 1024;
    uint dklenP1 = 1;
    uint dklenP2 = 1;

    
    thread uint8_t pbkdf2_1_out[MAX_PBKDF2_OUT_LEN];
    thread uint8_t romix_out[32];
    thread uint8_t pbkdf2_2_out[MAX_PBKDF2_OUT_LEN];


    // First PBKDF2 operation
    pbkdf2(ctx, block, block_len, dklenP1, pbkdf2_1_out);

    // // Endian conversion
    endian_full(pbkdf2_1_out, MAX_PBKDF2_OUT_LEN);

    // // Romix operation
    romix(pbkdf2_1_out, N, romix_out);

    // // Endian conversion
    endian_full(romix_out, 32);

    // // Second PBKDF2 operation
    pbkdf2_2nd(ctx, romix_out, 32, block, block_len, dklenP2, pbkdf2_2_out);

    // // Copy result to output buffer
    for (uint i = 0; i < MAX_PBKDF2_OUT_LEN; i++) {
        scrypt_out[i] = pbkdf2_2_out[i];
    }
}


kernel void scrypt_kernel(
    device uint8_t *inputs,
    device uint8_t *outputs,
    uint thread_position_in_grid [[thread_position_in_grid]]
) {

    uint32_t input_length = 80;
    uint32_t output_length = 32;

    // calculate input_start based on thread_position_in_grid
    uint32_t input_start = thread_position_in_grid * input_length;

    // calculate output_start based on thread_position_in_grid
    uint32_t output_start = thread_position_in_grid * output_length;

    // device variables
    device uint8_t *device_input = inputs + input_start;
    device uint8_t *device_output = outputs + output_start;

    thread sha256_context ctx;
    thread uint8_t thread_input[80];
    thread uint8_t thread_output[32];

     // copy device input to thread input
    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = device_input[i];
    }

    scrypt(&ctx, thread_input, input_length, thread_output);

    for (uint i = 0; i < output_length; i++) {
        device_output[i] = thread_output[i];
    }
}


