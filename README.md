# ZOracle

# Oracle Custom External Plugin
The purpose of this External Plugin is to provide the ability to write custom querys on zabbix server size.   
All querys are explicit in Template Macros.   
Template was created based on the official Oracle Template.   
Querys have been rewritten to work on older Oracle Database versions (tested on 11g).   
Items have been recreated as discovered items to be able to support Oracle RAC.   








**Original Oracle Template**
[Link to Original Oracle template.](https://git.zabbix.com/projects/ZBX/repos/zabbix/browse/templates/db/oracle_agent2) 

## Requirements
* Zabbix Agent 2
* Go >= 1.13 (required only to build from source)
* Oracle Instant Client >= 12 (or any client that works with Go and is compatible with your database version)

## Supported versions
* Oracle Database 11g, 12c 
* Possibly others :)

## Configuration
The Zabbix agent 2 configuration file is used to configure plugins.
      
## Supported keys
**oracle.custom.query[<commonParams\>,query[,args...]]** — Returns result of a custom query.  
*Parameters:*  
query (required) — sql query to execute.  
args (optional) — one or more arguments to pass to a query.

So, you can execute them:
  
    zoracle.custom.query[<commonParams>,'select 0 from dual']  
    zoracle.custom.query[<commonParams>,'SELECT amount FROM payment WHERE user = :1 AND service_id = :2 AND date = :3',"John Doe",1,"10/25/2020"]
          
You can pass as many parameters to a query as you need.   
The syntax for placeholder parameters uses ":#", where "#" is an index number of a parameter.   

Best approarch is to create a macro with the query string and use the macro on item key.


**oracle.ping[<commonParams\>]** — Tests if connection is alive or not.  
*Returns:*
- "1" if a connection is alive.
- otherwise return the error as string. Ex: ORA-12545: Connect failed because target host or object does not exist.
