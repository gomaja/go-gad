FROM golang:1.27.2-bookworm@sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61
RUN apt-get update && apt-get install -y --no-install-recommends build-essential autoconf automake libtool pkg-config libtalloc-dev curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN curl --fail --location --retry 3 https://api.github.com/repos/osmocom/libosmocore/tarball/73fed749b1e0ccb8f55226adab0f023561e51ed3 -o /tmp/osmo.tar.gz && echo '901ab0d956548da130424913fa7de37debe2ea03a20a45cc72973735c1fe61e3  /tmp/osmo.tar.gz' | sha256sum -c - && mkdir /build && tar xzf /tmp/osmo.tar.gz -C /build --strip-components=1 && rm /tmp/osmo.tar.gz
WORKDIR /build
RUN printf '1.14.2\n' > .tarball-version && autoreconf -fi && ./configure --disable-uring --disable-pcsc --disable-libusb --disable-gnutls --disable-systemd-logging --disable-libmnl --disable-libsctp --disable-sctp-tests --disable-uring-tests --disable-doxygen --disable-utilities --disable-gb && make -j4 && make install && ldconfig
COPY osmo.c /tmp/osmo.c
RUN cc -Wall -Wextra -Werror /tmp/osmo.c -o /usr/local/bin/gad-harness $(pkg-config --cflags --libs libosmogsm)
ENTRYPOINT ["gad-harness"]
