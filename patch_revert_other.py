import re

with open("internal/fs/inode/file.go", "r") as f:
    content = f.read()

# The comment 3097390702 says "I do not think this makes much sense. Will revisit this later in the final PR."
# on line 1223 which corresponds to `f.recordFallback(openMode, metrics.WriteFallbackReasonOtherAttr)`
# Let's remove `f.recordFallback(openMode, metrics.WriteFallbackReasonOtherAttr)`
content = re.sub(
    r'(\t\tf\.recordFallback\(openMode, metrics\.WriteFallbackReasonOtherAttr\)\n)',
    r'',
    content
)

with open("internal/fs/inode/file.go", "w") as f:
    f.write(content)
