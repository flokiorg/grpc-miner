// code is from https://github.com/ilvn/SHA256

#include <metal_stdlib>

using namespace metal;


#define NULL nullptr

constant const uint32_t K[64] = {
    0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5,
    0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
    0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3,
    0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
    0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc,
    0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
    0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
    0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
    0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13,
    0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
    0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3,
    0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
    0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5,
    0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
    0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208,
    0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2
};

struct sha256_context {
    uint8_t  buf[64];
    uint32_t hash[8];
    uint32_t bits[2];
    uint32_t len;
    uint32_t rfu__;
    uint32_t W[64];
};

#define _Ch(x, y, z) ((x & y) ^ ((~x) & z))
#define _Ma(x, y, z) ((x & y) ^ (x & z) ^ (y & z))
#define _shb(x, n) ((x >> (n & 31)) & 0xff)
#define _shw(x, n) ((x << (n & 31)) & 0xffffffff)
#define _r(x, n) ((x >> n) | _shw(x, 32 - n))
#define _S0(x) (_r(x, 2) ^ _r(x, 13) ^ _r(x, 22))
#define _S1(x) (_r(x, 6) ^ _r(x, 11) ^ _r(x, 25))
#define _G0(x) (_r(x, 7) ^ _r(x, 18) ^ (x >> 3))
#define _G1(x) (_r(x, 17) ^ _r(x, 19) ^ (x >> 10))
#define _word(c) (_shw((c)[0], 24) | _shw((c)[1], 16) | _shw((c)[2], 8) | ((c)[3]))
#define _addbits(ctx, n)                                 \
    do {                                                 \
        if ((ctx)->bits[0] > (0xffffffff - (n))) {       \
            (ctx)->bits[1] = ((ctx)->bits[1] + 1) & 0xFFFFFFFF; \
        }                                                \
        (ctx)->bits[0] = ((ctx)->bits[0] + (n)) & 0xFFFFFFFF;   \
    } while(0)
    
void _hash(thread sha256_context *ctx)
{
    uint32_t a, b, c, d, e, f, g, h;
    uint32_t t[2];

    a = ctx->hash[0];
    b = ctx->hash[1];
    c = ctx->hash[2];
    d = ctx->hash[3];
    e = ctx->hash[4];
    f = ctx->hash[5];
    g = ctx->hash[6];
    h = ctx->hash[7];

    for (uint32_t i = 0; i < 64; i++) {
        if (i < 16) {
            ctx->W[i] = _word((&ctx->buf[_shw(i, 2)]));
        } else {
            ctx->W[i] = _G1(ctx->W[i - 2])  + ctx->W[i - 7] +
                        _G0(ctx->W[i - 15]) + ctx->W[i - 16];
        }

        t[0] = h + _S1(e) + _Ch(e, f, g) + K[i] + ctx->W[i];
        t[1] = _S0(a) + _Ma(a, b, c);
        h = g;
        g = f;
        f = e;
        e = d + t[0];
        d = c;
        c = b;
        b = a;
        a = t[0] + t[1];
    }

    ctx->hash[0] += a;
    ctx->hash[1] += b;
    ctx->hash[2] += c;
    ctx->hash[3] += d;
    ctx->hash[4] += e;
    ctx->hash[5] += f;
    ctx->hash[6] += g;
    ctx->hash[7] += h;
} // _hash

void sha256_init(thread sha256_context *ctx)
{
    if (ctx != NULL) {
        ctx->bits[0] = ctx->bits[1] = ctx->len = 0;
        ctx->hash[0] = 0x6a09e667;
        ctx->hash[1] = 0xbb67ae85;
        ctx->hash[2] = 0x3c6ef372;
        ctx->hash[3] = 0xa54ff53a;
        ctx->hash[4] = 0x510e527f;
        ctx->hash[5] = 0x9b05688c;
        ctx->hash[6] = 0x1f83d9ab;
        ctx->hash[7] = 0x5be0cd19;
    }
} // sha256_init


void sha256_init_withState(thread sha256_context *ctx, device uint32_t *state)
{
    ctx->bits[0] = 512;
    ctx->bits[1] = 0;

    ctx->len = 0;

    for (uint32_t i = 0; i < 8; i++) {
        ctx->hash[i] = state[i];
    }
} // sha256_init_withState

void sha256_hash(thread sha256_context *ctx, const thread uint8_t *data, size_t len)
{
    if ((ctx != NULL) && (data != NULL) && (ctx->len < sizeof(ctx->buf))) {
        for (size_t i = 0; i < len; i++) {
            ctx->buf[ctx->len++] = data[i];
            if (ctx->len == sizeof(ctx->buf)) {
                _hash(ctx);
                _addbits(ctx, sizeof(ctx->buf) * 8);
                ctx->len = 0;
            }
        }
    }
} // sha256_hash

void sha256_done(thread sha256_context *ctx, thread uint8_t *output)
{
    uint32_t i, j;

    if (ctx != NULL) {
        j = ctx->len % sizeof(ctx->buf);
        ctx->buf[j] = 0x80;
        for (i = j + 1; i < sizeof(ctx->buf); i++) {
            ctx->buf[i] = 0x00;
        }

        if (ctx->len > 55) {
            _hash(ctx);
            for (j = 0; j < sizeof(ctx->buf); j++) {
                ctx->buf[j] = 0x00;
            }
        }

        _addbits(ctx, ctx->len * 8);
        ctx->buf[63] = _shb(ctx->bits[0],  0);
        ctx->buf[62] = _shb(ctx->bits[0],  8);
        ctx->buf[61] = _shb(ctx->bits[0], 16);
        ctx->buf[60] = _shb(ctx->bits[0], 24);
        ctx->buf[59] = _shb(ctx->bits[1],  0);
        ctx->buf[58] = _shb(ctx->bits[1],  8);
        ctx->buf[57] = _shb(ctx->bits[1], 16);
        ctx->buf[56] = _shb(ctx->bits[1], 24);
        _hash(ctx);

        if (output != NULL) {
            for (i = 0, j = 24; i < 4; i++, j -= 8) {
                output[i +  0] = _shb(ctx->hash[0], j);
                output[i +  4] = _shb(ctx->hash[1], j);
                output[i +  8] = _shb(ctx->hash[2], j);
                output[i + 12] = _shb(ctx->hash[3], j);
                output[i + 16] = _shb(ctx->hash[4], j);
                output[i + 20] = _shb(ctx->hash[5], j);
                output[i + 24] = _shb(ctx->hash[6], j);
                output[i + 28] = _shb(ctx->hash[7], j);
            }
        }
    }
} // sha256_done

void sha256_done_32(thread sha256_context *ctx, thread uint32_t *output) {
    uint32_t i, j;

    if (ctx != NULL) {
        // Padding the message
        j = ctx->len % sizeof(ctx->buf);
        ctx->buf[j] = 0x80;
        for (i = j + 1; i < sizeof(ctx->buf); i++) {
            ctx->buf[i] = 0x00;
        }

        // Process an additional block if needed
        if (ctx->len > 55) {
            _hash(ctx);
            for (j = 0; j < sizeof(ctx->buf); j++) {
                ctx->buf[j] = 0x00;
            }
        }

        // Append the length in bits and process the final block
        _addbits(ctx, ctx->len * 8);
        ctx->buf[63] = _shb(ctx->bits[0],  0);
        ctx->buf[62] = _shb(ctx->bits[0],  8);
        ctx->buf[61] = _shb(ctx->bits[0], 16);
        ctx->buf[60] = _shb(ctx->bits[0], 24);
        ctx->buf[59] = _shb(ctx->bits[1],  0);
        ctx->buf[58] = _shb(ctx->bits[1],  8);
        ctx->buf[57] = _shb(ctx->bits[1], 16);
        ctx->buf[56] = _shb(ctx->bits[1], 24);
        _hash(ctx);

        // Output the final hash as uint32_t values
        if (output != NULL) {
            for (i = 0; i < 8; i++) {
                output[i] = ctx->hash[i];
            }
        }
    }
} // sha256_done

void uint32ArrayToBytes(thread uint32_t *input, thread uint8_t *output, uint len) {
    for (uint i = 0; i < len; i++) {
        uint32_t value = input[i];
        
        output[i * 4 + 0] = (value >> 24) & 0xFF; // Most significant byte // littlenedian
        output[i * 4 + 1] = (value >> 16) & 0xFF;
        output[i * 4 + 2] = (value >> 8)  & 0xFF;
        output[i * 4 + 3] = value & 0xFF;          // Least significant byte
    }
}

void sha256(thread sha256_context *ctx,  thread uint8_t *input, size_t len, thread uint8_t *output)
{
    sha256_init(ctx);
    sha256_hash(ctx, input, len);
    sha256_done(ctx, output);
} // sha256



void double_sha256(thread sha256_context *ctx,  thread uint8_t *input, size_t len, thread uint8_t *output)
{
    sha256(ctx, input, len, output);
    sha256(ctx, output, 32, output);
} // double sha256


void sha256_words(thread sha256_context *ctx,  thread uint8_t *input, size_t len, thread uint32_t *output)
{
    sha256_init(ctx);
    sha256_hash(ctx, input, len);
    sha256_done_32(ctx, output);
} // sha256_words


void sha256_words_2(thread sha256_context *ctx,  thread uint32_t *input, size_t len, thread uint32_t *output)
{
    sha256_init(ctx);

    thread uint8_t middle_output[200]; 
    uint32ArrayToBytes(input, middle_output, len);

    sha256_hash(ctx, middle_output, len*4);

    sha256_done_32(ctx, output);
} // sha256_words

void double_sha256_words(thread sha256_context *ctx,  thread uint8_t *input, size_t len, thread uint32_t *output)
{
    sha256_words(ctx, input, len, output);

    thread uint8_t middle_input[32]; 
    uint32ArrayToBytes(output, middle_input, 8);

    sha256_words(ctx, middle_input, 32, output);
} // double sha256 words


void sha256_words_withState(device uint32_t *state, thread sha256_context *ctx, const thread uint8_t *input, size_t len, thread uint32_t *output)
{
    sha256_init_withState(ctx, state);
    sha256_hash(ctx, input, len);
    sha256_done_32(ctx, output);
} // sha256 withState


void fast_double_sha256_words(device uint32_t *state, thread sha256_context *ctx,  thread uint8_t *input, size_t len, thread uint32_t *output)
{
    sha256_words_withState(state, ctx, input, len, output);

    thread uint8_t middle_inputs[32]; 
    uint32ArrayToBytes(output, middle_inputs, 8);

    sha256_words(ctx, middle_inputs, 32, output);
} // double sha256 words

///////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////
///////////////////////////////////////////////////////////////////////////////////////////////////



kernel void test_sha256_bytes_kernel(
    device uint8_t *inputs,
    device uint8_t *outputs
) {

    const uint input_length = 80;
    const uint output_length = 32;

    thread sha256_context ctx;
    thread uint8_t thread_input[input_length];
    thread uint8_t thread_output[output_length];

    for (uint32_t i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }
  
    sha256(&ctx, thread_input, sizeof(thread_input), thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}


kernel void test_doubleSha256_bytes_kernel(
    device uint8_t *inputs,
    device uint8_t *outputs
) {

    const uint input_length = 80;
    const uint output_length = 32;

    thread sha256_context ctx;
    thread uint8_t thread_input[input_length];
    thread uint8_t thread_output[output_length];

    for (uint32_t i = 0; i < input_length; i++) {
        thread_input[i] = inputs[i];
    }
  
    double_sha256(&ctx, thread_input, sizeof(thread_input), thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}

kernel void test_sha256_words_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 20;
    const uint output_length = 8;

    thread sha256_context ctx;
    
    thread uint32_t thread_input_32[input_length];
    thread uint8_t thread_input[input_length*4];
    thread uint32_t thread_output[output_length];
    
    for (uint32_t i = 0; i < input_length; i++) {
        thread_input_32[i] = inputs[i];
    }

    uint32ArrayToBytes(thread_input_32, thread_input, input_length);

  
    sha256_words(&ctx, thread_input, sizeof(thread_input), thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}

kernel void test_sha256_words_2_kernel(
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
  
    sha256_words_2(&ctx, thread_input, input_length, thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}

kernel void test_doubleSha256_words_kernel(
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 20;
    const uint output_length = 8;

    thread sha256_context ctx;

    thread uint32_t thread_input_32[input_length];
    thread uint8_t thread_input[input_length*4];
    thread uint32_t thread_output[output_length];
    
    for (uint32_t i = 0; i < input_length; i++) {
        thread_input_32[i] = inputs[i];
    }

    uint32ArrayToBytes(thread_input_32, thread_input, input_length);
  
    double_sha256_words(&ctx, thread_input, sizeof(thread_input), thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}


kernel void test_fast_doubleSha256_words_kernel(
    device uint32_t *state,
    device uint32_t *inputs,
    device uint32_t *outputs
) {

    const uint input_length = 4;
    const uint output_length = 8;

    thread sha256_context ctx;

    thread uint32_t thread_input_32[input_length];
    thread uint8_t thread_input[input_length*4];
    thread uint32_t thread_output[output_length];
    
    for (uint32_t i = 0; i < input_length; i++) {
        thread_input_32[i] = inputs[i];
    }

    uint32ArrayToBytes(thread_input_32, thread_input, input_length);
  
    fast_double_sha256_words(state, &ctx, thread_input, sizeof(thread_input), thread_output);

    for (uint i = 0; i < output_length; i++) {
        outputs[i] = thread_output[i];
    }
}

uint8_t countRightZeros(thread uint32_t *items, int len) {
    uint8_t totalCount = 0;

    for (int i = len-1; i >= 0; i--) {
        uint32_t value = items[i];
        
        if (value == 0) {
            totalCount += 32;
            continue;
        }

        uint8_t count = 0;
        while ((value & 1) == 0) {
            count++;
            value >>= 1;
        }

        totalCount += count;
        break;
    }

    return totalCount;
}

kernel void sha256_kernel(device uint32_t *state,
                          device uint32_t *inputs,
                          device uint8_t *outputs,
                          uint thread_position_in_grid [[ thread_position_in_grid ]])
{
    const uint input_length = 4;
    const uint output_length = 8;

    uint32_t input_start = thread_position_in_grid * input_length;
    uint32_t output_start = thread_position_in_grid;

    device uint32_t *device_input = inputs + input_start;
    device uint8_t *device_output = outputs + output_start;

    
    thread sha256_context ctx;
    thread uint32_t thread_input_32[input_length];
    thread uint8_t thread_input[input_length*4];
    thread uint32_t thread_output[output_length];

    for (uint32_t i = 0; i < input_length; i++) {
        thread_input_32[i] = device_input[i];
    }

    uint32ArrayToBytes(thread_input_32, thread_input, input_length);

    fast_double_sha256_words(state, &ctx, thread_input, sizeof(thread_input), thread_output);

    *device_output = countRightZeros(thread_output, 8);

}

