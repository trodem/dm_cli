### 1. Comandi Base e Ciclo di Vita dei Container
Questi comandi gestiscono le operazioni fondamentali sui container:
*   **`docker run`**: Crea e avvia un nuovo container da un'immagine.
*   **`docker create`**: Crea un nuovo container senza avviarlo.
*   **`docker start`**: Avvia uno o più container fermi.
*   **`docker stop`**: Arresta in modo graduale (graceful) un container in esecuzione.
*   **`docker restart`**: Combina i comandi di stop e start.
*   **`docker kill`**: Arresta forzatamente un container inviando un segnale SIGKILL.
*   **`docker rm`**: Rimuove uno o più container (usa l'opzione `-f` per rimuovere quelli in esecuzione).
*   **`docker pause` / `unpause`**: Sospende o riprende tutti i processi all'interno di un container.

### 2. Ispezione e Interazione
Per monitorare cosa succede nei container e interagirvi:
*   **`docker ps`**: Elenca i container in esecuzione (usa `-a` per vedere anche quelli fermi).
*   **`docker logs`**: Mostra l'output (stdout e stderr) di un container.
*   **`docker exec`**: Esegue un comando in un container già avviato; è utile per il debug (es. `docker exec -it <nome> bash`).
*   **`docker inspect`**: Estrae informazioni dettagliate a basso livello sui container o immagini.
*   **`docker top`**: Visualizza i processi in esecuzione all'interno di un container.
*   **`docker cp`**: Copia file o cartelle tra il container e il file system locale.
*   **`docker attach`**: Si collega ai flussi standard (input, output, errore) di un container in esecuzione.

### 3. Gestione delle Immagini
Comandi per creare, scaricare e organizzare le immagini:
*   **`docker images`**: Elenca tutte le immagini locali.
*   **`docker build`**: Crea un'immagine a partire da un **Dockerfile**.
*   **`docker pull` / `push`**: Scarica o carica un'immagine da/verso un registro (come Docker Hub).
*   **`docker rmi`**: Elimina una o più immagini locali.
*   **`docker tag`**: Assegna un tag (un nome specifico) a un'immagine.
*   **`docker history`**: Mostra la cronologia degli strati (layers) di un'immagine.
*   **`docker save` / `load`**: Esporta o importa un'immagine in formato tarball.

### 4. Docker Compose
Per gestire applicazioni multi-container definite in un file `docker-compose.yml`:
*   **`docker compose up`**: Crea e avvia i servizi definiti nel file.
*   **`docker compose down`**: Ferma e rimuove container, reti e immagini definiti nel file.
*   **`docker compose ps`**: Elenca i container gestiti dal file Compose.
*   **`docker compose start` / `stop` / `pause` / `unpause`**: Gestisce lo stato dei servizi.

### 5. Pulizia e Manutenzione
*   **`docker image prune`**: Rimuove le immagini inutilizzate o "dangling".
*   **`docker system prune`**: Pulisce l'intero sistema rimuovendo container fermi, reti inutilizzate e immagini senza tag.

### 6. Funzioni Avanzate (Sicurezza e AI)
*   **Docker Scout**: Usato per l'analisi delle vulnerabilità (es. `docker scout cves IMAGE`).
*   **Docker Model Runner**: Permette di scaricare ed eseguire modelli di AI (es. `docker model pull ai/smollm2`, `docker model run MODEL_NAME "prompt"`).
*   **Docker MCP**: Supporta il Model Context Protocol per connettere assistenti AI a fonti dati esterne (es. `docker mcp server list`).

Ti interessa approfondire una di queste categorie o vuoi che crei un **quiz** per testare la tua conoscenza dei comandi Docker?
