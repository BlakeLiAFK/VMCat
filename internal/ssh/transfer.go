package ssh

import (
	"fmt"
	"io"
	"strings"
)

// WriteFile 通过 SSH 将数据写入远程文件，支持进度回调
func (c *Client) WriteFile(remotePath string, reader io.Reader, size int64, onProgress func(written int64)) error {
	c.mu.Lock()
	if c.client == nil || c.closed {
		c.mu.Unlock()
		if err := c.Connect(); err != nil {
			return err
		}
		c.mu.Lock()
	}
	client := c.client
	c.mu.Unlock()

	session, err := client.NewSession()
	if err != nil {
		if reconnErr := c.Connect(); reconnErr != nil {
			return fmt.Errorf("reconnect: %w", reconnErr)
		}
		c.mu.Lock()
		client = c.client
		c.mu.Unlock()
		session, err = client.NewSession()
		if err != nil {
			return fmt.Errorf("new session: %w", err)
		}
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	// 确保目标目录存在
	dir := remotePath[:strings.LastIndex(remotePath, "/")]
	if err := session.Start(fmt.Sprintf("mkdir -p '%s' && cat > '%s'", dir, remotePath)); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	buf := make([]byte, 64*1024)
	var written int64
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if _, writeErr := stdin.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write: %w", writeErr)
			}
			written += int64(n)
			if onProgress != nil {
				onProgress(written)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read: %w", readErr)
		}
	}

	stdin.Close()
	if err := session.Wait(); err != nil {
		return fmt.Errorf("wait: %w", err)
	}
	return nil
}
