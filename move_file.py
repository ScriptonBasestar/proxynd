#!/usr/bin/env python3
import os
import shutil

# Remove the problematic file
try:
    os.remove("health/mocks/repository.go")
    print("File removed successfully")
except Exception as e:
    print(f"Error removing file: {e}")

# Copy new file over
try:
    shutil.move("health/mocks/repository_new.go", "health/mocks/repository.go")
    print("File moved successfully")
except Exception as e:
    print(f"Error moving file: {e}")