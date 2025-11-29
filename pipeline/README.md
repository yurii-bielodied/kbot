Для виконання рекомендується декларативний формат Jenkins Pipeline: https://www.jenkins.io/doc/book/pipeline/

Приклад виконання:

Встановіть Kind та Kubernetes на ваш локальний комп'ютер.
Створіть кластер Kubernetes за допомогою Kind:
kind create cluster --name jenkins
      
Встановіть Jenkins на кластер Kubernetes за допомогою Helm:
helm repo add jenkinsci https://charts.jenkins.io/
helm repo update
helm install jenkins jenkinsci/jenkins
      
Після запуску Jenkins отримайте доступ до інтерфейсу користувача Jenkins за допомогою наступної команди:
kubectl port-forward svc/jenkins 8080:8080
      
Тепер ви можете отримати доступ до Jenkins за адресою у вашому веббраузері:
http://localhost:8080
      
Завдання:

Візьміть за основу Jenkins Groovy приклад:
https://github.com/den-vasyliev/kbot/blob/main/pipeline/jenkins.groovy
  
Налаштуйте Jenkins Pipeline для створення параметризованої збірки за використанням pipeline скрипту з вашого репозиторію як зображено на прикладі:
https://github.com/den-vasyliev/kbot/blob/main/pipeline/jenkins-pipeline.png
  
Розробник повинен мати можливість вибрати параметри збірки або використати налаштування по замовчуванню, наприклад:
https://github.com/den-vasyliev/kbot/blob/main/pipeline/jenkins-job.png
