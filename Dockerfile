FROM fedora:latest
ADD journald-cloudwatch-logs .

CMD echo "state_file = \"/var/run/journald-cloudwatch-state\"" > /etc/journald-cloudwatch && \
    echo "journal_dir = \"/journal/\"" >> /etc/journald-cloudwatch && \
    echo "log_group = \"$LOG_GROUP\"" >> /etc/journald-cloudwatch && \
    echo "aws_region = \"$AWS_REGION\"" >> /etc/journald-cloudwatch && \
    echo "log_stream = \"journal\"" >> /etc/journald-cloudwatch && \
    ./journald-cloudwatch-logs /etc/journald-cloudwatch

