1) What I learned today?

  Today I learned how to classify errors in which the program can find corrupted JSON files,it can also determine bad hashes,empty blocks,duplicate blocks.I now know to use JSON flag and how to implement it in the code.

Features Added:-

✅ 1. Corrupted JSON - Detects invalid JSON format
✅ 2. Bad Hash - Hash doesn't match computed SHA256
✅ 3. Timestamp Future - Block timestamp in the future (>5 min tolerance)
✅ 4. Timestamp Past - Timestamp too old (>10 years)
✅ 5. Timestamp Not Increasing - Timestamps don't increase
✅ 6. Duplicate Hashes - Same hash appears multiple times
✅ 7. Empty Blocks - Blocks with no data
✅ 8. PrevHash Errors - Broken chain linkage
✅ 9. Height Errors - Block height mismatches
✅ 10. Missing Blocks - Gaps in the chain
✅ 11. Out of Order - Blocks not in sequence