ARG UBI_VERSION=9.5-1739420147
FROM redhat/ubi9-minimal:${UBI_VERSION}

USER root

# Install required packages
RUN microdnf install -y tar gzip && \
    microdnf clean all

# Create ghost user and group
RUN groupadd -g 99 ghost && \
    useradd -u 99 -g ghost -m -s /bin/bash ghost && \
    # Create and set permissions for workspace directory
    mkdir -p /workspace && \
    chown ghost:ghost /workspace && \
    chmod 755 /workspace

# Set working directory
WORKDIR /workspace

# Switch to ghost user
USER ghost

CMD ["sleep", "infinity"]
