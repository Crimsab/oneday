use std::process::{Child, Command};

#[cfg(unix)]
pub struct ProcessContainment {
    process_group: libc::pid_t,
}

#[cfg(unix)]
pub fn configure(command: &mut Command) {
    use std::os::unix::process::CommandExt;
    command.process_group(0);
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

#[cfg(windows)]
pub fn configure(_command: &mut Command) {}

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
        if job == 0 {
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
        if process == 0 {
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
    #[cfg(unix)]
    #[test]
    fn unix_signals_target_the_whole_process_group() {
        assert_eq!(super::group_signal_target(42), -42);
    }
}
