#!/usr/bin/env python3
"""
Simple RAW RGBA file analyzer without external dependencies
"""

import sys
import struct

def analyze_raw_rgba(filename, width=256, height=240):
    """Analyze RAW RGBA file"""
    try:
        # Read raw RGBA data
        with open(filename, 'rb') as f:
            data = f.read()
        
        print(f"File: {filename}")
        print(f"Size: {len(data)} bytes")
        
        # Expected size check
        expected_size = width * height * 4
        if len(data) == expected_size:
            print(f"✓ Size matches {width}x{height} RGBA")
        else:
            # Try to detect actual dimensions
            total_pixels = len(data) // 4
            print(f"⚠ Size mismatch. Expected {expected_size}, got {len(data)}")
            print(f"Total pixels: {total_pixels}")
            
            # Common resolutions
            if total_pixels == 768 * 720:  # 768x720 (256*3 x 240*3)
                width, height = 768, 720
                print(f"Detected resolution: {width}x{height} (scaled)")
            elif total_pixels == 256 * 240:
                width, height = 256, 240
                print(f"Detected resolution: {width}x{height} (original)")
        
        # Analyze first few pixels
        print("\nFirst 8 pixels (RGBA):")
        for i in range(min(8, len(data) // 4)):
            offset = i * 4
            r, g, b, a = struct.unpack('BBBB', data[offset:offset+4])
            print(f"  Pixel {i}: R={r:3d} G={g:3d} B={b:3d} A={a:3d} (#{r:02X}{g:02X}{b:02X})")
        
        # Analyze color distribution (sample every 1000th pixel)
        colors = {}
        step = max(1, len(data) // (4 * 1000))  # Sample ~1000 pixels
        for i in range(0, len(data) - 3, step * 4):
            r, g, b, a = struct.unpack('BBBB', data[i:i+4])
            color = (r, g, b, a)
            colors[color] = colors.get(color, 0) + 1
        
        print(f"\nColor distribution (sampled):")
        sorted_colors = sorted(colors.items(), key=lambda x: x[1], reverse=True)
        for i, ((r, g, b, a), count) in enumerate(sorted_colors[:10]):
            print(f"  Color {i+1}: R={r:3d} G={g:3d} B={b:3d} A={a:3d} (#{r:02X}{g:02X}{b:02X}) - {count} samples")
        
        # Check if this looks like test pattern (should have red, green, blue, white)
        expected_colors = {
            (255, 0, 0, 255),    # Red
            (0, 255, 0, 255),    # Green  
            (0, 0, 255, 255),    # Blue
            (255, 255, 255, 255) # White
        }
        
        found_colors = set(colors.keys())
        matches = expected_colors.intersection(found_colors)
        
        if len(matches) == 4:
            print(f"\n✓ Test pattern detected! Found all 4 expected colors")
        else:
            print(f"\n⚠ Test pattern check: Found {len(matches)}/4 expected colors")
            missing = expected_colors - found_colors
            if missing:
                print("  Missing colors:")
                for r, g, b, a in missing:
                    print(f"    R={r:3d} G={g:3d} B={b:3d} A={a:3d}")
        
        return True
        
    except Exception as e:
        print(f"❌ Error analyzing {filename}: {e}")
        return False

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 simple_analyze.py <raw_file1> [raw_file2] ...")
        return
    
    for filename in sys.argv[1:]:
        print("=" * 60)
        analyze_raw_rgba(filename)
        print()

if __name__ == "__main__":
    main()