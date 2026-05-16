#!/usr/bin/env python3
import sys

def analyze_frame(filename):
    """Analyze a raw framebuffer to look for text patterns"""
    try:
        with open(filename, 'rb') as f:
            data = f.read()
        
        print(f"Analyzing {filename} ({len(data)} bytes)")
        
        # Count different pixel values
        pixel_counts = {}
        for i in range(0, len(data), 4):  # Assuming RGBA format
            if i + 3 < len(data):
                pixel = data[i:i+4]
                key = tuple(pixel)
                pixel_counts[key] = pixel_counts.get(key, 0) + 1
        
        print(f"Found {len(pixel_counts)} unique pixel values:")
        for pixel, count in sorted(pixel_counts.items(), key=lambda x: x[1], reverse=True)[:10]:
            print(f"  RGBA({pixel[0]:02X},{pixel[1]:02X},{pixel[2]:02X},{pixel[3]:02X}): {count} pixels")
        
        # Look for patterns that might indicate text
        non_zero_pixels = sum(1 for i in range(0, len(data), 4) if any(data[i:i+4]))
        print(f"Non-zero pixels: {non_zero_pixels} / {len(data)//4}")
        
    except Exception as e:
        print(f"Error analyzing {filename}: {e}")

if __name__ == "__main__":
    if len(sys.argv) > 1:
        analyze_frame(sys.argv[1])
    else:
        analyze_frame("debug_frame_600.raw")