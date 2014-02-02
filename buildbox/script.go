package buildbox

import (
  "fmt"
  "time"
  "github.com/kr/pty"
  "os/exec"
  "io"
  "os"
  "bytes"
  "log"
  "path"
  "path/filepath"
  "errors"
  "syscall"
)

type Process struct {
  Output string
  Pid int
  Running bool
  ExitStatus int
  Success bool
}

// Implement the Stringer thingy
func (p Process) String() string {
  return fmt.Sprintf("Process{Pid: %d, Running: %t, ExitStatus: %d}", p.Pid, p.Running, p.ExitStatus)
}

func RunScript(dir string, script string, env []string, callback func(Process)) (*Process, error) {
  // Create a new instance of our process struct
  var process Process

  // Find the script to run
  absoluteDir, _ := filepath.Abs(dir)
  pathToScript := path.Join(absoluteDir, script)

  log.Printf("Running: %s from within %s\n", script, absoluteDir)

  command := exec.Command(pathToScript)
  command.Dir = absoluteDir

  // Copy the current processes ENV and merge in the
  // new ones. We do this so the sub process gets PATH
  // and stuff.
  // TODO: Is this correct?
  currentEnv := os.Environ()
  command.Env = append(currentEnv, env...)

  // Start our process
  pty, err := pty.Start(command)
  if err != nil {
    return nil, err
  }

  process.Pid = command.Process.Pid
  process.Running = true

  var buffer bytes.Buffer

  go func() {
    // Copy the pty to our buffer. This will block until it EOF's
    // or something breaks.
    // TODO: What if this fails?
    io.Copy(&buffer, pty)
  }()

  go func(){
    for process.Running {
      // Convert the stdout buffer to a string
      process.Output = buffer.String()

      // Call the callback and pass in our process object
      callback(process)

      // Sleep for 1 second
      time.Sleep(1000 * time.Millisecond)
    }
  }()

  // Wait until the process has finished
  waitResult := command.Wait()

  // Update the process with the final results
  // of the script
  process.Running = false
  process.ExitStatus = getExitStatus(waitResult)
  process.Output = buffer.String()
  process.Success = true

  // No error occured so we can return nil
  return &process, nil
}

// https://github.com/hnakamur/commango/blob/fe42b1cf82bf536ce7e24dceaef6656002e03743/os/executil/executil.go#L29
// WTF? Computers... (shrug)
// TODO: Can this be better?
func getExitStatus(waitResult error) int {
  if waitResult != nil {
    if err, ok := waitResult.(*exec.ExitError); ok {
      if s, ok := err.Sys().(syscall.WaitStatus); ok {
        return s.ExitStatus()
      } else {
        panic(errors.New("Unimplemented for system where exec.ExitError.Sys() is not syscall.WaitStatus."))
      }
    }
  }
  return 0
}