#!/usr/bin/env python3
"""
RAW RGBA files analyzer for debugging SDL2 color issues
"""

import sys
import numpy as np
from PIL import Image

def analyze_raw_rgba(filename, width=256, height=240):
    """Analyze RAW RGBA file and convert to PNG"""
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
        
        # Reshape data
        pixels = np.frombuffer(data, dtype=np.uint8)
        if len(pixels) % 4 != 0:
            print("❌ Invalid RGBA data length")
            return
            
        pixels = pixels.reshape((-1, 4))[:width*height]  # Take only what we need
        pixels = pixels.reshape((height, width, 4))
        
        # Analyze first few pixels
        print("\nFirst 8 pixels (RGBA):")
        for i in range(min(8, width)):
            r, g, b, a = pixels[0, i]
            print(f"  Pixel {i}: R={r:3d} G={g:3d} B={b:3d} A={a:3d} (#{r:02X}{g:02X}{b:02X})")
        
        # Analyze color distribution
        unique_colors = np.unique(pixels.reshape(-1, 4), axis=0)
        print(f"\nUnique colors: {len(unique_colors)}")
        for i, (r, g, b, a) in enumerate(unique_colors[:10]):  # Show first 10
            count = np.sum(np.all(pixels.reshape(-1, 4) == [r, g, b, a], axis=1))
            print(f"  Color {i+1}: R={r:3d} G={g:3d} B={b:3d} A={a:3d} (#{r:02X}{g:02X}{b:02X}) - {count} pixels")
        
        # Convert to PIL Image and save as PNG
        img = Image.fromarray(pixels[:, :, :3], 'RGB')  # Drop alpha for PNG
        png_filename = filename.replace('.raw', '.png')
        img.save(png_filename)
        print(f"\n✓ Converted to PNG: {png_filename}")
        
        return True
        
    except Exception as e:
        print(f"❌ Error analyzing {filename}: {e}")
        return False

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 analyze_raw.py <raw_file1> [raw_file2] ...")
        print("Example: python3 analyze_raw.py test_pattern.raw nes_frame.raw")
        return
    
    for filename in sys.argv[1:]:
        print("=" * 60)
        analyze_raw_rgba(filename)
        print()

if __name__ == "__main__":
    main()
