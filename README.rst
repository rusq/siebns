==============
siebns Package
==============

Package siebns currently only allows fixing the encoded file size in Oracle
`Siebel CRM`_ Gateway Naming file after making manual modifications to it.

It provides the NSFile structure and member functions to allow loading and
fixing the aforementioned files.

Example::
    
  var ns = siebns.NSFile{}

  err := ns.Load("siebns.dat")
  if err != nil {
      log.Fatalf("%s", err)
  }
  defer ns.Close()

  if ns.CorrectionNeeded {
      wrote, err := ns.FixSize()
      if err != nil {
          log.Fatalf("Error writing to file:  %s\n", err)
      }
  }

Please consult the package documentation for further details.

.. _`Siebel CRM`: http://www.oracle.com/us/products/applications/siebel/overview/index.html