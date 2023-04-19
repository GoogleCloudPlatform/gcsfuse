# Here $1 is referring to first argument which is testBucket
echo "This is from directory Test file a" >> a.txt
gsutil mv a.txt gs://$1/Test/
echo "This is from file Test1" >> Test1.txt
gsutil mv Test1.txt gs://$1/
echo "This is from directory Test/b file b" >> b.txt
gsutil mv b.txt gs://$1/Test/b/
