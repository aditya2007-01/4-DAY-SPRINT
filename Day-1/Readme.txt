1) What I understood today?
   How to implement and verify the data of blocks in database. I also understood how to implement the code for finding missing blocks,duplicate blocks,timestamps anomalies & out of order blocks.

Which Lines contain what code 

✅ 1. Validate hash == SHA256(blockData) - Line 142-158
✅ 2. Validate prevHash link end-to-end - Line 170-199
✅ 3. Detect missing blocks - Line 126-139
✅ 4. Detect duplicate hashes - Line 160-174
✅ 5. Detect height mismatches - Line 201-213
✅ 6. Detect timestamp anomalies - Line 215-247
✅ 7. Detect out-of-order blocks - Line 249-260