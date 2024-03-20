let jsMailerBase = "https://jsmailer.example.com"

function jsMailerGetToken() {
    if(window.hasForm === true) {
        let formId = window.formId
        let formElem = document.querySelector(window.formElem)
        if(typeof formElem === 'undefined' || formElem === null || formId === null) {
            console.error("Required values not set. Need formId and formElem")
            return false
        }

        // Fetch a token
        let formData = new FormData()
        formData.append("formid", formId)
        fetch(jsMailerBase + '/api/v1/token', {
            mode: 'cors',
            method: 'post',
            body: formData
        }).then(response => {
            if(!response.ok) {
                if(response.status === 401) {
                    let submitBtn = formElem.querySelector('#submitButton')
                    if(typeof submitBtn !== 'undefined' && submitBtn !== null) {
                        submitBtn.classList.remove("btn-primary")
                        submitBtn.classList.add('btn-secondary')
                        submitBtn.style.display = 'none'
                    }
                    showError("This page is not authorized to use the form mailer system.")
                }
            }
            return response.json()
        }).then(respJson => {
            if(respJson !== null) {
                formElem.action = respJson.data.url
                formElem.method = respJson.data.method
                formElem.enctype = respJson.data.enc_type
            }
        }).catch(() => {
            let submitBtn = formElem.querySelector('#submitButton')
            if(typeof submitBtn !== 'undefined' && submitBtn !== null) {
                submitBtn.classList.remove("btn-primary")
                submitBtn.classList.add('btn-secondary')
                submitBtn.style.display = 'none'
            }
            showError("Failed to fetch security token from form mailer system")
            return false
        })
    }
}

function jsMailerSendMail(clickEvent, formElem) {
    let isValid = formElem.checkValidity();
    if(isValid === true) {
        clickEvent.preventDefault()
        let submitBtn = formElem.querySelector('#submitButton')
        if(typeof submitBtn !== 'undefined' && submitBtn !== null) {
            submitBtn.disabled = true
            submitBtn.innerText = 'Sende Nachricht...';
            let formData = new FormData(formElem)
            fetch(formElem.action, {
                mode: 'cors',
                method: 'post',
                body: formData
            }).then(response => {
                return response.json()
            }).then(respJson => {
                if(respJson !== null) {
                    formElem.querySelectorAll('input').forEach(field => {
                        field.value = ''
                    })
                    formElem.querySelectorAll('textarea').forEach(field => {
                        field.value = ''
                    })
                    submitBtn.disabled = true
                    submitBtn.classList.remove('btn-primary')
                    submitBtn.classList.add('btn-success')
                    submitBtn.innerText = 'Thanks! Your message has been sent.'
                    return true
                } else {
                    showError("An error occurred sending your message.")
                    submitBtn.classList.remove("btn-primary")
                    submitBtn.classList.add('btn-secondary')
                    submitBtn.innerText = 'Error!'
                    return false
                }
            }).catch(() => {
                showError("An error occurred sending your message.")
                submitBtn.classList.remove("btn-primary")
                submitBtn.classList.add('btn-secondary')
                submitBtn.innerText = 'Error!'
                return false
            })
        }
    }
}

function showError(errorMsg) {
    let msgDiv = document.querySelector('#errorMsg')
    if(typeof msgDiv === 'undefined' || msgDiv === null) {
        return false
    }
    msgDiv.innerText = errorMsg
    msgDiv.innerHTML = encodeURIComponent(msgDiv.innerText) + '<button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>'
    msgDiv.style.display = 'block'
}