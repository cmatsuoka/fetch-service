The Go import redirector inspector
==================================

The Go import redirector inspector examines redirection responses to
requests for go modules. The response is an HTML document containing
a ``meta`` tag with ``name`` set to ``go-import``.

Inspector ID
------------

``go.import-redirector``

Internal state
--------------

None

Request verification
--------------------

The current implementation allows downloads from common servers hosting
git modules. The request URL must have the parameter ``go-get`` set to 1.

Acceptance criteria
-------------------

To be accepted, the artefact must have:

* The MIME type set to "text/html".
* The HTML meta tag with name set to ``go-import``.

Rejection reasons
-----------------

* The artefact cannot be parsed as an HTML document.
* The ``go-import`` meta tag is not set.

Extracted metadata
------------------

The following pieces of metadata are extracted by the inspector:

.. table:: Go import redirector inspector metadata
   :widths: auto

   ============  ====  ============================================
   Field         Used  Data source
   ============  ====  ============================================
   type          Yes   ``text/html``
   name          Yes   ``Go import redirector``
   version
   description   Yes   ``HTML file wit go-import meta tag``
   vendor        Yes   The server hostname
   author
   author-email
   architecture
   license
   copyright
   ============  ====  ============================================
