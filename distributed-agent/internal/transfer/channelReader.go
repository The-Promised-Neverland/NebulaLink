package transfer

import "io"

// channelReader implements io.Reader by reading streamed byte chunks from channels.
// Will allow consumers (e.g. tar/gzip readers, io.Copy) to process chunked data as a continuous byte stream.
type channelReader struct {
	dataCh <-chan []byte
	errCh  <-chan error
	buffer []byte
}

func (r *channelReader) Read(p []byte) (n int, err error) {
	if len(r.buffer) > 0 {
		n = copy(p, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}
	select {
	case chunk, ok := <-r.dataCh:
		if !ok {
			select {
			case err := <-r.errCh:
				return 0, err
			default:
				return 0, io.EOF
			}
		}
		n = copy(p, chunk)
		if n < len(chunk) {
			r.buffer = chunk[n:]
		}
		return n, nil
	case err := <-r.errCh:
		return 0, err
	}
}
