using namespace metal;


void hmac(
    thread sha256_context *ctx,

    thread uint32_t *salt,
    size_t salt_len,

    thread uint32_t *message,
    size_t message_len,

    thread uint32_t *output
) {

    uint32_t khash[8];  // Holds the result of the first hash (SHA-256 produces 32 bytes)
    sha256_words_2(ctx, message, message_len, khash);

    // Define IPAD and OPAD for 64-byte block size
    const uint32_t IPAD = 0x36363636;
    const uint32_t OPAD = 0x5c5c5c5c;

    uint32_t ixor[16];  // Holds the XORed iPad result (64 bytes)
    uint32_t oxor[16];  // Holds the XORed oPad result (64 bytes)

    // XOR khash with IPAD and OPAD to create inner and outer padded keys
    for (size_t i = 0; i < 8; i++) {
        ixor[i] = IPAD ^ khash[i];
        oxor[i] = OPAD ^ khash[i];
    }
    for (size_t i = 8; i < 16; i++) {
        ixor[i] = IPAD;
        oxor[i] = OPAD;
    }

    // Create input for the inner hash by combining ixor and salt
    uint32_t in_ihash[40];  // 16 words for ixor + 20 words for salt
    for (size_t i = 0; i < 16; i++) {
        in_ihash[i] = ixor[i];
    }
    for (size_t i = 0; i < salt_len; i++) {
        in_ihash[16 + i] = salt[i];
    }


    uint32_t ihash[8];  // Result of the inner hash
    sha256_words_2(ctx, in_ihash, 16 + salt_len, ihash);


    // Create input for the outer hash by combining oxor and ihash
    uint32_t in_ohash[24];  // 16 words for oxor + 8 words for ihash
    for (size_t i = 0; i < 16; i++) {
        in_ohash[i] = oxor[i];
    }
    for (size_t i = 0; i < 8; i++) {
        in_ohash[16 + i] = ihash[i];
    }

    sha256_words_2(ctx, in_ohash, 24, output);
}

kernel void test_hmac_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 20;
    const uint output_length = 8;

    thread sha256_context ctx;
    thread uint32_t thread_input[input_length];
    thread uint32_t thread_output[output_length];

    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }

    hmac(&ctx, thread_input, input_length, thread_input, input_length, thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}

kernel void test_hmac_plus_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 20;
    const uint output_length = 8;

    thread sha256_context ctx;
    thread uint32_t thread_input[input_length];
    thread uint32_t thread_output[output_length];

    for (uint i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }

    thread uint32_t salt[input_length+1];
    for (uint i = 0; i < input_length; i++) {
        salt[i] = inputs[i];
    }
    salt[input_length] = 1;

    hmac(&ctx, salt, input_length+1, thread_input, input_length, thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}