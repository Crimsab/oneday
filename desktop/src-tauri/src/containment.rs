use std::process::{Child, Command};

#[cfg(unix)]
pub struct ProcessContainment {
    process_group: libc::pid_t,
}

#[cfg(unix)]
pub fn configure(command: &mut Command) {
    use std::os::unix::process::CommandExt;
    command.process_group(0);

    #[cfg(target_os = "linux")]
    {
        let desktop_pid = unsafe { libc::getpid() };
        unsafe {
            command.pre_exec(move || {
                if libc::prctl(libc::PR_SET_PDEATHSIG, libc::SIGKILL) == -1 {
                    return Err(std::io::Error::last_os_error());
                }
                // PR_SET_PDEATHSIG is installed after fork. If the desktop
                // exited in that short window, fail the child before exec so
                // it can never become an orphaned gateway.
                if libc::getppid() != desktop_pid {
                    libc::_exit(1);
                }
                Ok(())
            });
        }
    }
}

#[cfg(unix)]
impl ProcessContainment {
    pub fn attach(child: &Child) -> Result<Self, String> {
        Ok(Self {
            process_group: child.id() as libc::pid_t,
        })
    }

    pub fn request_graceful_stop(&self) -> Result<(), String> {
        signal_group(self.process_group, libc::SIGINT)
    }

    pub fn force_stop(&self) -> Result<(), String> {
        signal_group(self.process_group, libc::SIGKILL)
    }
}

#[cfg(unix)]
fn signal_group(process_group: libc::pid_t, signal: libc::c_int) -> Result<(), String> {
    let result = unsafe { libc::kill(group_signal_target(process_group), signal) };
    if result == 0 || std::io::Error::last_os_error().raw_os_error() == Some(libc::ESRCH) {
        Ok(())
    } else {
        Err(format!(
            "Could not signal the local OneDay process group: {}",
            std::io::Error::last_os_error()
        ))
    }
}

#[cfg(unix)]
fn group_signal_target(process_group: libc::pid_t) -> libc::pid_t {
    -process_group
}

#[cfg(windows)]
pub struct ProcessContainment {
    job: windows_sys::Win32::Foundation::HANDLE,
}

// A Windows job handle may be transferred between threads. It is owned by this
// value, only accessed through the LocalProcess mutex, and closed exactly once
// in Drop.
#[cfg(windows)]
unsafe impl Send for ProcessContainment {}

#[cfg(windows)]
pub fn configure(command: &mut Command) {
    use std::os::windows::process::CommandExt;
    use windows_sys::Win32::System::Threading::CREATE_NO_WINDOW;

    command.creation_flags(CREATE_NO_WINDOW);
}

#[cfg(windows)]
impl ProcessContainment {
    pub fn attach(child: &Child) -> Result<Self, String> {
        use std::mem::{size_of, zeroed};
        use std::ptr::null;
        use windows_sys::Win32::Foundation::{CloseHandle, HANDLE};
        use windows_sys::Win32::System::JobObjects::{
            AssignProcessToJobObject, CreateJobObjectW, JobObjectExtendedLimitInformation,
            SetInformationJobObject, JOBOBJECT_EXTENDED_LIMIT_INFORMATION,
            JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
        };
        use windows_sys::Win32::System::Threading::{
            OpenProcess, PROCESS_QUERY_LIMITED_INFORMATION, PROCESS_SET_QUOTA, PROCESS_TERMINATE,
        };

        let job = unsafe { CreateJobObjectW(null(), null()) };
        if job.is_null() {
            return Err(format!(
                "Could not create local process job: {}",
                std::io::Error::last_os_error()
            ));
        }
        let mut limits: JOBOBJECT_EXTENDED_LIMIT_INFORMATION = unsafe { zeroed() };
        limits.BasicLimitInformation.LimitFlags = JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE;
        let configured = unsafe {
            SetInformationJobObject(
                job,
                JobObjectExtendedLimitInformation,
                &limits as *const _ as *const _,
                size_of::<JOBOBJECT_EXTENDED_LIMIT_INFORMATION>() as u32,
            )
        };
        if configured == 0 {
            unsafe { CloseHandle(job) };
            return Err(format!(
                "Could not configure local process job: {}",
                std::io::Error::last_os_error()
            ));
        }
        let process: HANDLE = unsafe {
            OpenProcess(
                PROCESS_SET_QUOTA | PROCESS_TERMINATE | PROCESS_QUERY_LIMITED_INFORMATION,
                0,
                child.id(),
            )
        };
        if process.is_null() {
            unsafe { CloseHandle(job) };
            return Err(format!(
                "Could not open local gateway process: {}",
                std::io::Error::last_os_error()
            ));
        }
        let assigned = unsafe { AssignProcessToJobObject(job, process) };
        unsafe { CloseHandle(process) };
        if assigned == 0 {
            unsafe { CloseHandle(job) };
            return Err(format!(
                "Could not contain local gateway process: {}",
                std::io::Error::last_os_error()
            ));
        }
        Ok(Self { job })
    }

    pub fn request_graceful_stop(&self) -> Result<(), String> {
        Ok(())
    }

    pub fn force_stop(&self) -> Result<(), String> {
        use windows_sys::Win32::System::JobObjects::TerminateJobObject;
        let stopped = unsafe { TerminateJobObject(self.job, 1) };
        if stopped == 0 {
            return Err(format!(
                "Could not stop local process job: {}",
                std::io::Error::last_os_error()
            ));
        }
        Ok(())
    }
}

#[cfg(windows)]
impl Drop for ProcessContainment {
    fn drop(&mut self) {
        unsafe { windows_sys::Win32::Foundation::CloseHandle(self.job) };
    }
}

#[cfg(test)]
mod tests {
    #[cfg(target_os = "linux")]
    #[test]
    fn linux_children_request_kill_when_the_desktop_exits() {
        use std::time::{Duration, Instant};

        let output = std::process::Command::new(std::env::current_exe().expect("test binary"))
            .args(["--ignored", "linux_parent_death_helper", "--nocapture"])
            .output()
            .expect("launch parent-death helper");
        assert!(output.status.success());
        let stdout = String::from_utf8(output.stdout).expect("helper output");
        let child_pid: libc::pid_t = stdout
            .lines()
            .find_map(|line| line.strip_prefix("ONEDAY_CHILD_PID="))
            .expect("helper child pid")
            .parse()
            .expect("numeric helper child pid");

        let deadline = Instant::now() + Duration::from_secs(2);
        while Instant::now() < deadline {
            let state = std::fs::read_to_string(format!("/proc/{child_pid}/stat"))
                .ok()
                .and_then(|stat| stat.rsplit_once(") ").map(|(_, fields)| fields.to_owned()))
                .and_then(|fields| fields.split_whitespace().next().map(str::to_owned));
            if state.as_deref().is_none_or(|state| state == "Z") {
                return;
            }
            std::thread::sleep(Duration::from_millis(25));
        }

        unsafe { libc::kill(child_pid, libc::SIGKILL) };
        panic!("contained child survived its desktop parent");
    }

    #[cfg(target_os = "linux")]
    #[test]
    #[ignore = "helper process for linux_children_request_kill_when_the_desktop_exits"]
    #[allow(clippy::zombie_processes)] // The helper must exit without reaping to exercise PDEATHSIG.
    fn linux_parent_death_helper() {
        use std::io::Write;
        use std::process::{Command, Stdio};

        let mut command = Command::new("sleep");
        command
            .arg("30")
            .stdin(Stdio::null())
            .stdout(Stdio::null())
            .stderr(Stdio::null());
        super::configure(&mut command);
        let child = command.spawn().expect("spawn contained child");
        println!("ONEDAY_CHILD_PID={}", child.id());
        std::io::stdout().flush().expect("flush child pid");
    }

    #[cfg(unix)]
    #[test]
    fn unix_signals_target_the_whole_process_group() {
        assert_eq!(super::group_signal_target(42), -42);
    }
}
