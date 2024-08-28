# filesync

Simple CLI tool that will sync the folder it's ran inside of. The tool syncs the folder
to the location provided in the command at the host provided. Under the hood the tool uses
rsync and assumes that you have a key setup in your ssh config to allow SSH access to the
destination machine. The tool also assumes that rsync is installed on both the source
and destination machines.

**DISCLAIMER:** This tool is first and foremost a tool for my own workflow. There are countless improvements that could be added but if it doesn't affect my workflow chances are I won't spend the time to add requested features.

Due to the nature of the tool being for my own work flow, the underlying `rsync` command
looks like the following
```bash
rsync -avz . {server}:{folder_dest} --include=**.gitignore --exclude=/.git --filter=:- .gitignore --delete-after

```
## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
- [Contributing](#contributing)
- [License](#license)
- [Features](#features)
- [Roadmap](#roadmap)

## <a name="installation"></a>Installation

### Installing with go
For easiest installation, use `go install` to install the tool to your go environment.
```bash
# Example installation commands if applicable
go install github.com/CryoViking/filesync@latest
```

### Installing from source
You can also choose to build from source
```bash
git clone https://github.com/CryoViking/filesync.git
cd filesync
go get -u -v all
go build
go install
```

## <a name="usage"></a>Usage

Instructions on how to use the project, including code examples if necessary.
```bash
filesync help
```
Example:
```bash
filesync --ssh-address "devbox" /home/dev/TestLab
```
## <a name="contributing"></a>Contributing

If this project blows up then I'll figure out a proper process for contributing.  
For now tho just follow the following steps (normal for most projects afaik)

1. Fork the repository and clone it to your local machine.
2. Create a new branch for your feature or bug fix: `git checkout -b feature-branch-name`.
3. Make your changes and ensure they are well-tested.
4. Commit your changes: `git commit -m "Description of your changes"`.
5. Push to the branch: `git push origin feature-branch-name`.
6. Submit a pull request to the `master` branch of the original repository.
7. Ensure your pull request includes a clear description of the changes made and any relevant information for reviewers.
8. After submitting your pull request, a project maintainer (me for now) will review your changes and provide feedback.
9. Once your changes are approved, they will be merged into the `master` branch and become part of the project.

## <a name="license"></a>License

This project is licensed under the MIT License - see the [LICENSE](https://github.com/CryoViking/filesync/blob/master/LICENSE) file for details.

## <a name="features"></a>Features

A brief description of the features of this project.

- Utilises `rsync` under the hood to transfer the files.
- Simple CLI usage.

## <a name="roadmap"></a>Roadmap (IDK)
Yeah look... I don't know if I'll keep working on this. I primarily built this for my own use so 
as I get annoyed with things I'll make changes and push.
