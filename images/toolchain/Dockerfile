FROM python:3.13.14-slim-bookworm@sha256:de572b33eae61a53675a87bbd02b5e365df7b6b2b06c9276124e965cec08c452
WORKDIR /work
COPY requirements.txt /tmp/image-requirements.txt
RUN python -m pip install --disable-pip-version-check --no-cache-dir --require-hashes --only-binary=:all: -r /tmp/image-requirements.txt
ENTRYPOINT ["python", "-B", "images/generate.py"]
