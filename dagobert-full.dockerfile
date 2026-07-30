FROM sprungknoedl/dagobert:latest AS app

# ---------------------------------

FROM log2timeline/plaso
USER root
ADD https://github.com/Yamato-Security/hayabusa/releases/download/v3.2.0/hayabusa-3.2.0-lin-x64-musl.zip /usr/src/hayabusa.zip

RUN apt update && apt install -y ca-certificates unzip
RUN unzip /usr/src/hayabusa.zip -d /opt/hayabusa && \
    mv /opt/hayabusa/hayabusa-3.2.0-lin-x64-musl /opt/hayabusa/hayabusa && \
    chmod +x /opt/hayabusa/hayabusa

COPY --from=app /home/sprungknoedl/dagobert /home/plaso/dagobert
COPY --from=app /home/sprungknoedl/external/mitre /home/plaso/external/mitre
COPY --from=app /home/sprungknoedl/docker-entrypoint.sh /home/plaso/docker-entrypoint.sh
RUN chmod +x /home/plaso/docker-entrypoint.sh

ENV MODULE_PLASO="/usr/bin/psteal.py --unattended"
ENV MODULE_HAYABUSA="/opt/hayabusa/hayabusa"
ENV PATH="$PATH:/home/plaso"

WORKDIR /home/plaso
# MITRE data is already baked in (copied above) so this only fetches
# Hayabusa's rule set via its UpdateAssets hook, now that MODULE_HAYABUSA is
# set; also creates a throwaway data/dagobert.db here, not copied into the
# final image.
RUN dagobert update

# entrypoint bootstraps a fresh data volume (migrate db) before exec-ing dagobert
ENTRYPOINT ["/home/plaso/docker-entrypoint.sh"]
CMD ["server"]
