Dagobert enhances incident investigations by supporting external processing modules like Hayabusa for EVTX analysis and Plaso for timeline generation. These modules operate through worker instances that handle evidence independently. The platform offers three flexible approaches to module execution, catering to different operational requirements.

## 🐳 Pre-Bundled Docker Containers (Recommended Approach)
For immediate productivity, Dagobert provides complete Docker containers that combine both the worker service and processing tools. These ready-to-deploy packages offer the simplest implementation path, ensuring proper tool compatibility and isolation. While convenient, this method does require Docker and creates separate container instances for each module. The trade-off involves less control over tool versions, as they're fixed during the container's creation.

✅ **Pros:**
- **Zero setup** – Runs out of the box 🚀
- **Perfect compatibility** – Tools tested together 🔒   
- **Isolated & secure** – No host system mess 🧼

⚠️ **Notes:**
- Needs Docker 🐋
- One container per module (slightly heavier) 📦
- Tool versions fixed at build time 🔧

💡 **Perfect for:** Teams who want **fast, hassle-free** processing!

📝 **Configuration:**
This Docker Compose configuration deploys Dagobert with pre-bundled processing modules, utilizing a shared volume for coordinated evidence handling. The setup consists of four services: the primary Dagobert application (exposed on port 8080) and three specialized worker containers (Hayabusa, Plaso, and Timesketch). 

All components share a persistent `data` volume mounted at `/home/sprungknoedl/files`, ensuring seamless evidence transfer between services. 

Each worker container is configured with parallel processing capabilities through the `DAGOBERT_WORKERS`environment variable, and connects to the main application using credentials specified in the shared `dagobert.env` file (`DAGOBERT_URL` and `DAGOBERT_API_KEY`).

Example `docker-compose.yml`:
```yaml
volumes:
  data:

services:
  app:
    image: sprungknoedl/dagobert
    restart: unless-stopped
    volumes:
      - data:/home/sprungknoedl/files
    env_file: dagobert.env
    ports:
      - 8080:8080

  worker-hayabusa:
    image: sprungknoedl/dagobert-hayabusa
    restart: unless-stopped
    depends_on: [app]
    volumes:
      - data:/home/sprungknoedl/files
    env_file: dagobert.env
    environment:
      - DAGOBERT_WORKERS=3

  worker-plaso:
    image: sprungknoedl/dagobert-plaso
    restart: unless-stopped
    depends_on: [app]
    volumes:
      - data:/home/plaso/files
    env_file: dagobert.env
    environment:
      - DAGOBERT_WORKERS=5

  worker-timesketch:
    image: sprungknoedl/dagobert-timesketch
    restart: unless-stopped
    depends_on: [app]
    volumes:
      - data:/home/sprungknoedl/files
    env_file: dagobert.env
    environment:
      - DAGOBERT_WORKERS=3
```

## 💻 Local Execution Mode
When direct system access is preferred, workers can utilize tools installed natively on the host machine. This approach grants investigators full control over tool versions and signature databases without intermediate layers. However, it lacks process isolation and requires manual maintenance of all dependencies.

✅ **Pros:**
- **Full version control** – Update tools anytime ⬆️
- **Direct access** – No middleman 🏎
- **Works offline** – No Docker needed ❌🐳

⚠️ **Notes:**
- **No isolation** – Tools run on the host system 🏠
- Manual dependency management 🔄

💡 **Perfect for:** Teams who **love customization** and don’t mind maintenance!

📝 **Configuration:**

For local module execution, Dagobert workers use system-installed tools specified through environment variables. Configure the paths to each module's executable using:
- `MODULE_PLASO` for Plaso's psteal
- `MODULE_HAYABUSA` for Hayabusa
- `MODULE_TIMESKETCH` for Timesketch's importer

Example configuration (`dagobert.env`):
```env
MODULE_PLASO=psteal
MODULE_HAYABUSA=hayabusa
MODULE_TIMESKETCH=timesketch_importer
```

To launch a worker instance 🚀:
```sh
dagobert worker
```

Each worker requires two essential environment variables for coordination:
- `DAGOBERT_URL`: Main node connection endpoint
- `DAGOBERT_API_KEY`: Authentication credential

Optionally control processing capacity with:
- `DAGOBERT_WORKERS`: Number of parallel jobs (default: 3)

This configuration provides direct access to host-installed tools while maintaining integration with Dagobert's task distribution system. Ensure all specified executables are in the system `PATH` or provide full absolute paths.

## 📦 Dockerized Tool Execution
A middle-ground solution involves workers launching tools in ephemeral Docker containers. This preserves the isolation benefits of containerization while allowing more version flexibility than pre-bundled containers. Implementers should note this configuration requires appropriate Docker permissions and creates some additional operational overhead compared to direct local execution.

✅ **Pros:**
- **Clean & secure** – Tools run in containers 🧊
- **Version freedom** – Swap tool images easily 🏷️

⚠️ **Notes:**
- Needs Docker permissions 🔑
- Slightly slower than local mode 🐢

💡 **Perfect for:** Teams who want **isolation without locking versions**!

📝 **Configuration:**

For containerized module execution, Dagobert workers invoke tools through Docker, running each process in an isolated and ephemeral environment. Configure the Docker commands to use for each module using:

- **`MODULE_PLASO`** for Plaso (Docker image `log2timeline/plaso`)
- **`MODULE_HAYABUSA`** for Hayabusa (Docker image `sprungknoedl/hayabusa` )
- **`MODULE_TIMESKETCH`** for Timesketch (Docker image `sprungknoedl/timesketch-importer`)

Example configuration (`dagobert.env`):
```env 
MODULE_PLASO=docker run -v $PWD/files:/home/plaso/files log2timeline/plaso psteal  
MODULE_HAYABUSA=docker run -v $PWD/files:/home/sprungknoedl/files sprungknoedl/hayabusa  
MODULE_TIMESKETCH=docker run -v $PWD/files:/home/sprungknoedl/files sprungknoedl/timesketch-importer  
```

‼️ **Key Requirements:**
- **Volume Mounting**: Each container must have the shared file location mounted relative to the containers working folder as `files` (e.g. `/home/plaso/files`).    
- **Docker Runtime**: The host system must have Docker installed and the worker process must have permission to execute `docker run`.

To start the worker 🚀:
```sh
dagobert worker
```

As with local execution, the worker requires:
- **`DAGOBERT_URL`**: Address of the main coordination node.
- **`DAGOBERT_API_KEY`**: Authentication key for secure registration.
- **`DAGOBERT_WORKERS`** _(optional)_: Number of concurrent jobs (default: 3).

This approach combines tool isolation with Dagobert’s task distribution, ensuring reproducible execution while maintaining workflow integration.

## 📡 Worker Registration
During startup, each worker instance performs automatic capability verification. It systematically checks the availability of all configured modules through their respective interfaces. Only modules passing these validation checks become active and visible to the main Dagobert instance. This ensures the coordination node only dispatches tasks to properly configured workers, maintaining system reliability. The self-registration process happens transparently, requiring no manual intervention beyond proper initial setup.

The platform's flexible architecture allows security teams to mix these approaches across different modules or even deploy multiple workers with varying configurations. This adaptability ensures organizations can tailor the system to their specific security posture and infrastructure constraints while maintaining consistent investigative workflows.