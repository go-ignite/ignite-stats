FROM alpine
MAINTAINER Wendell Sun <iwendellsun@gmail.com>

WORKDIR /ignite-stats
ADD ignite-stats /ignite-stats/ignite-stats

RUN echo '*/5 * * * * /ignite-stats/ignite-stats -c /ignite-stats/config.toml -m instant' > crontab.tmp \
  && echo '0 0 * * * /ignite-stats/ignite-stats -c /ignite-stats/config.toml -m daily' >> crontab.tmp \
  && echo '0 0 1 * * /ignite-stats/ignite-stats -c /ignite-stats/config.toml -m monthly' >> crontab.tmp \
		&& crontab crontab.tmp \
  && rm -rf crontab.tmp

ENTRYPOINT ["/bin/sh", "-c", "/usr/sbin/crond -f -d 0"]
