# Proctoring and privacy

The Python service accepts explicit signals: camera availability, face count, coarse face-position movement, tab visibility, and an attempt identifier. It returns a candidate suspicious event with a confidence score and evidence metadata. It does not upload or persist raw video and it does not label a student as cheating.

The browser implementation must request camera permission transparently, show a preview, run a preflight check, display the monitoring state during the exam, tolerate reconnection, and send only the minimum metadata needed for review. Production deployment requires a privacy impact assessment, a retention schedule, access controls, student notice and an appeal/review process.
