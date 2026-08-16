package vm

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/auth"
)

// Read straight from /proc rather than shelling out to ps, which busybox
// provides in a cut-down form here.
const (
	procRoot     = "/proc"
	maxProcesses = 40
)

// Processes the device needs in order to keep working. Killing kvm_system
// stops video capture and the OLED; killing the server takes the web UI down
// with it, including the button that was just pressed.
var protectedNames = map[string]bool{
	"init":           true,
	"NanoKVM-Server": true,
	"kvm_system":     true,
}

func (s *Service) GetProcesses(c *gin.Context) {
	var rsp proto.Response

	entries, err := os.ReadDir(procRoot)
	if err != nil {
		log.Errorf("failed to read %s: %s", procRoot, err)
		rsp.ErrRsp(c, -1, "failed to read processes")
		return
	}

	pageSize := readPageSize()
	memTotal, _ := readMemory()

	processes := make([]proto.ProcessInfo, 0, 64)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		process, ok := readProcess(pid, pageSize, memTotal)
		if !ok {
			continue
		}
		processes = append(processes, process)
	}

	// Heaviest first: the point of the list is finding what is consuming the
	// device, not enumerating everything on it.
	sort.Slice(processes, func(i, j int) bool {
		if processes[i].MemoryBytes != processes[j].MemoryBytes {
			return processes[i].MemoryBytes > processes[j].MemoryBytes
		}
		return processes[i].Pid < processes[j].Pid
	})

	if len(processes) > maxProcesses {
		processes = processes[:maxProcesses]
	}

	rsp.OkRspWithData(c, &proto.GetProcessesRsp{Processes: processes})
}

// KillProcess sends SIGTERM, or SIGKILL when force is set.
func (s *Service) KillProcess(c *gin.Context) {
	var req proto.KillProcessReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if req.Pid <= 1 {
		// PID 1 is init; killing it panics the kernel.
		rsp.ErrRsp(c, -2, "refusing to kill this process")
		return
	}

	name := readProcessName(req.Pid)
	if name == "" {
		rsp.ErrRsp(c, -3, "process not found")
		return
	}
	if protectedNames[name] {
		rsp.ErrRsp(c, -4, fmt.Sprintf("%s keeps this device running and cannot be killed here", name))
		return
	}
	// Killing our own process group would take the UI down with it.
	if req.Pid == os.Getpid() {
		rsp.ErrRsp(c, -4, "refusing to kill this process")
		return
	}

	signal := syscall.SIGTERM
	if req.Force {
		signal = syscall.SIGKILL
	}

	if err := syscall.Kill(req.Pid, signal); err != nil {
		log.Errorf("failed to kill %d: %s", req.Pid, err)
		rsp.ErrRsp(c, -5, "failed to stop process")
		return
	}

	auth.Audit(c, "process_kill", log.Fields{"pid": req.Pid, "name": name, "force": req.Force})

	rsp.OkRsp(c)
}

func readProcess(pid int, pageSize int64, memTotal uint64) (proto.ProcessInfo, bool) {
	// /proc/<pid>/stat holds the name in parentheses, which may itself contain
	// spaces, so the fields after it are located from the last ')'.
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return proto.ProcessInfo{}, false
	}

	content := string(data)
	open := strings.IndexByte(content, '(')
	closeIdx := strings.LastIndexByte(content, ')')
	if open < 0 || closeIdx < 0 || closeIdx < open {
		return proto.ProcessInfo{}, false
	}

	name := content[open+1 : closeIdx]
	fields := strings.Fields(content[closeIdx+1:])
	// After the name come state, ppid, ... with utime at index 11 and stime at
	// 12 counting from the state field.
	if len(fields) < 22 {
		return proto.ProcessInfo{}, false
	}

	process := proto.ProcessInfo{
		Pid:       pid,
		Name:      name,
		State:     fields[0],
		Protected: protectedNames[name],
	}

	// rss is in pages, at index 21 after the state field.
	if rss, err := strconv.ParseInt(fields[21], 10, 64); err == nil {
		process.MemoryBytes = uint64(rss * pageSize)
		if memTotal > 0 {
			process.MemoryPercent = round1(float64(process.MemoryBytes) / float64(memTotal) * 100)
		}
	}

	if cmdline := readCmdline(pid); cmdline != "" {
		process.Command = cmdline
	}

	return process, true
}

func readProcessName(pid int) string {
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "stat"))
	if err != nil {
		return ""
	}

	content := string(data)
	open := strings.IndexByte(content, '(')
	closeIdx := strings.LastIndexByte(content, ')')
	if open < 0 || closeIdx < 0 || closeIdx < open {
		return ""
	}

	return content[open+1 : closeIdx]
}

// readCmdline gives the full invocation, which distinguishes several processes
// that share a name.
func readCmdline(pid int) string {
	data, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "cmdline"))
	if err != nil || len(data) == 0 {
		return ""
	}

	// Arguments are NUL separated, with a trailing NUL.
	parts := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	return strings.TrimSpace(strings.Join(parts, " "))
}

func readPageSize() int64 {
	size := int64(os.Getpagesize())
	if size <= 0 {
		return 4096
	}
	return size
}
