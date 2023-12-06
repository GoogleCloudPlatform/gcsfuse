# Originally written for read cache logs but might extend to generic GCSFuse logs
# This script takes gcsfuse logs in json format and parse them into the
# following format:
# { Start Timestamp,
#   fileInode:
#   objectGeneration
#   objectName,
#   endTimestamp,
#   chunks: {
#     {startTime,
#      endTime,
#      startRange:
#      endRange:
#      cacheHit:
#      },
#     {startTime,
#      endTime,
#      startRange:
#      endRange:
#      cacheHit:
#      },
# }
