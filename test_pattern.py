#!/usr/bin/env python3

def rijndael_gf_multiply(a, b):
    """Rijndael Galois Field multiplication (GF(2^8))"""
    result = 0
    while b:
        if b & 1:
            result ^= a
        a <<= 1
        if a & 0x100:
            a ^= 0x11B  # Rijndael polynomial
        b >>= 1
    return result & 0xFF

def generate_test_pattern():
    """Generate the test pattern used by mmc3bigchrram.nes"""
    pattern = []
    
    # The test uses specific values - let's generate first 16 bytes
    # Based on the log, we see values like 0x03, 0x05, 0x0F, 0x11, 0x33, 0x55, 0xFF
    initial_values = [0x03, 0x05, 0x0F, 0x11, 0x33, 0x55, 0xFF, 0x1A]
    
    # Generate pattern for first 16 bytes
    for i in range(16):
        if i < len(initial_values):
            pattern.append(initial_values[i])
        else:
            # Use a simple formula for remaining bytes
            pattern.append((pattern[i-1] * 3) & 0xFF)
    
    return pattern

def main():
    pattern = generate_test_pattern()
    print("Expected test pattern (first 16 bytes):")
    print(" ".join(f"{b:02X}" for b in pattern))
    
    print("\nPattern as seen in logs:")
    logged_pattern = [0x03, 0x05, 0x0F, 0x11, 0x33, 0x55, 0xFF, 0x1A, 
                     0x2E, 0x72, 0x96, 0xA1, 0xF8, 0x13, 0x35, 0x5F]
    print(" ".join(f"{b:02X}" for b in logged_pattern))

if __name__ == "__main__":
    main()